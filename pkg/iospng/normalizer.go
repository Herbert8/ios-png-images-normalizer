/*
  背景：iOS 打包后的 App 中的 PNG 图像与标准 PNG 图像格式不同，导致在非苹果系统中无法查看
  本工具用于将非标准化的 PNG 图像进行标准化。

  文件格式：
      1、PNG 文件头："\x89PNG\r\n\x1a\n"
      2、多个 PNG 数据块
  PNG 数据块结构，分四部分：
      1、0-3 字节：该数据块中，数据部分的长度
      2、4-7 字节：该数据块的块类型，如：CgBI、IHDR、iTXt、tEXt、iDOT、IDAT、IDAT、IEND 等
         其中：
              IHDR 块 包含 图片的 宽、高
              IDAT 中包含需要重新编码的数据
      3、数据块中的实际有效数据，长度为 0-3 字节的 32 位证书表示的长度
      4、数据的 CRC32 值

  思路：PNG 图像的格式，在文件头之后，是多个类型的数据块。需要对 IDAT 数据块进行处理：
      1、将每个 IDAT 数据块中的 有效数据（不包含块的头信息）进行合并
      2、对合并后的数据进行 zlib 解码
      3、对解码后的数据重新进行编排
      4、对编排后的数据进行 zlib 编码
      5、为重新编码后的数据重新生成 IDAT 数据块

  参考资料：
      PNG File Format
      https://docs.fileformat.com/image/png/

      隐写术之图片隐写
      https://zhuanlan.zhihu.com/p/62895080

      iPhone PNG Images Normalizer
      https://axelbrz.com/?mod=iphone-png-images-normalizer

      iPhone PNG Images Normalizer with a fix for multiple IDAT
      https://gist.github.com/urielka/3609051
*/

package iospng

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"os"
	"unsafe"
)

// PngFileHeader PNG 文件头
const PngFileHeader = "\x89PNG\r\n\x1a\n"

var ErrNotPNGFormat = errors.New("iospng: data is not in PNG format")

// ChunkHeader PNG 数据块的头信息
type ChunkHeader struct {
	ChunkLength uint32  // 数据块长度
	ChunkType   [4]byte // 数据块类型
}

// GetChunkType 获取数据块的类型
func (receiver *ChunkHeader) GetChunkType() string {
	return string(receiver.ChunkType[:])
}

// ImageSize 图像的尺寸（宽、长）
type ImageSize struct {
	Width  int32
	Height int32
}

// ImageChunk PNG 的数据块
type ImageChunk struct {
	header ChunkHeader
	data   []byte
	crc    uint32
}

// GetFullChunkLength 获取数据块的完整长度
// 包含以下几部分内容：
// 1、数据块的头：4 字节长度，4 字节类型
// 2、有效数据
// 3、crc 校验和
func (receiver *ImageChunk) GetFullChunkLength() uint {
	return uint(unsafe.Sizeof(receiver.header)) +
		uint(receiver.header.ChunkLength) +
		uint(unsafe.Sizeof(receiver.crc))
}

func (receiver *ImageChunk) GetCRC() uint32 {
	return receiver.crc
}

func (receiver *ImageChunk) GetData() []byte {
	var retData []byte
	retData = append(retData, receiver.data...)
	return retData
}

func (receiver *ImageChunk) GetHeader() ChunkHeader {
	return receiver.header
}

// tryGetImageSize IHDR 类型的块包含图像的尺寸信息
func (receiver *ImageChunk) tryGetImageSize() *ImageSize {
	retImgSize := &ImageSize{}
	// 如果数据块类型为 IHDR
	if receiver.header.GetChunkType() == "IHDR" {
		chunkDataReader := bytes.NewReader(receiver.data)
		// 读取尺寸
		_ = binary.Read(chunkDataReader, binary.BigEndian, retImgSize)
	} else {
		// 不是 IHDR 类型则返回 nil
		retImgSize = nil
	}
	return retImgSize
}

// SaveFullChunkToData 将 数据块 中的描述保存到字节数组
func (receiver *ImageChunk) SaveFullChunkToData() []byte {
	chunkHeaderBuf := bytes.Buffer{}
	chunkCRCBuf := bytes.Buffer{}

	// 数据块 头信息
	_ = binary.Write(&chunkHeaderBuf, binary.BigEndian, receiver.header)
	// 数据块 crc
	_ = binary.Write(&chunkCRCBuf, binary.BigEndian, receiver.crc)

	// 拼接数据
	var retData []byte
	// 拼接 数据块 头信息
	retData = append(retData, chunkHeaderBuf.Bytes()...)
	// 拼接 数据块 有效数据
	retData = append(retData, receiver.data...)
	// 拼接 数据块 crc
	retData = append(retData, chunkCRCBuf.Bytes()...)

	return retData
}

// Copy 深度复制 Chunk 对象
func (receiver *ImageChunk) Copy() *ImageChunk {
	copiedImgChunk := new(ImageChunk)
	copiedImgChunk.header = receiver.header
	copiedImgChunk.data = append(copiedImgChunk.data, receiver.data...)
	copiedImgChunk.crc = receiver.crc
	return copiedImgChunk
}

// 根据字节数组生成数据块对象
// 注意：字节数组的开始位置为块的起始，但字节数组中可能包含不止一个数据块的数据
// 读取时，会根据 块头 的描述来读取对应长度的字节
func newPngChunk(data []byte) *ImageChunk {
	retChunk := new(ImageChunk)

	// 读取 header
	pngDataReader := bytes.NewReader(data)
	_ = binary.Read(pngDataReader, binary.BigEndian, &retChunk.header)

	// 读取数据体
	headerSize := unsafe.Sizeof(retChunk.header)
	headerAndChunkSize := headerSize + uintptr(retChunk.header.ChunkLength)
	retChunk.data = data[headerSize:headerAndChunkSize]

	// 读取 crc
	crcData := data[headerAndChunkSize : headerAndChunkSize+unsafe.Sizeof(retChunk.crc)]
	_ = binary.Read(bytes.NewReader(crcData), binary.BigEndian, &retChunk.crc)

	return retChunk
}

// CheckPngFileDataHeader 判断 PNG 文件数据（包含 PNG 文件头）的文件头是否合法
func CheckPngFileDataHeader(pngFileData []byte) bool {
	return bytes.HasPrefix(pngFileData, []byte(PngFileHeader))
}

// CheckPngFileHeader 判断 PNG 文件的文件头是否合法
func CheckPngFileHeader(pngFilename string) (bool, error) {
	pngFileData, err := os.ReadFile(pngFilename)
	if err != nil {
		return false, err
	}
	return CheckPngFileDataHeader(pngFileData), nil
}

type PNGImage struct {
	imageSize   ImageSize
	imageChunks []*ImageChunk
}

// ParsePngFileData 将 PNG 文件数据（包含 PNG 文件头）解析为 数据块 数组
func ParsePngFileData(pngFileData []byte) (*PNGImage, error) {

	// 判断格式是否合法
	if !CheckPngFileDataHeader(pngFileData) {
		return nil, ErrNotPNGFormat
	}

	// 从文件头之后，开始读取数据
	pngData := pngFileData[len(PngFileHeader):]

	// 用于存放数据块的数组
	var pngChunks []*ImageChunk

	for {
		// 创建数据块
		imgChunk := newPngChunk(pngData)
		// 收集数据块
		pngChunks = append(pngChunks, imgChunk)
		// 重置读取数据的起始位置
		pngData = pngData[imgChunk.GetFullChunkLength():]
		// 判断是否读到数据结尾
		if len(pngData) == 0 {
			break
		}
	}

	// 创建 PNGImage 对象
	retPngImg := new(PNGImage)
	// 为 PNGImage 指定 相关 数据块
	retPngImg.imageChunks = pngChunks

	imgSize := retPngImg.getImageSize()
	if imgSize != nil {
		retPngImg.imageSize = *imgSize
	} else {
		retPngImg.imageSize = ImageSize{Width: -1, Height: -1}
	}

	return retPngImg, nil
}

// ParsePngFile 将 PNG 文件（包含 PNG 文件头）解析为 数据块 数组
func ParsePngFile(pngFile string) (*PNGImage, error) {
	pngFileData, err := os.ReadFile(pngFile)
	if err != nil {
		return nil, err
	}
	return ParsePngFileData(pngFileData)
}

// 获取图片尺寸
func (receiver *PNGImage) getImageSize() *ImageSize {
	var imgSize *ImageSize
	for _, imgChunk := range receiver.imageChunks {
		imgSize = imgChunk.tryGetImageSize()
		if imgSize != nil {
			break
		}
	}
	return imgSize
}

// GetImageSize 获取图片尺寸
func (receiver *PNGImage) GetImageSize() ImageSize {
	return receiver.imageSize
}

func (receiver *PNGImage) Normalize() (*PNGImage, error) {

	// 用于存储整理后的 ImageChunk 数组
	var imgChunks []*ImageChunk

	// 用于收集所有 IDAT 数据块中数据的数组
	var allIDAT []byte

	imgSize := receiver.getImageSize()

	// 遍历所有数据块
	for _, chunk := range receiver.imageChunks {
		chunkCopy := chunk.Copy()
		// 默认情况，数据块不跳过操作
		bSkip := false
		// 判断数据块类型
		switch chunkCopy.header.GetChunkType() {
		case "IDAT":
			// 跳过直接写入操作，先收集数据，用于后续统一操作
			bSkip = true
			// 收集 IDAT 数据
			allIDAT = append(allIDAT, chunkCopy.data...)
		case "CgBI":
			// 跳过 CgBI 数据块
			bSkip = true
		case "IEND": // IEND 一般是最后才出现，在这里重新处理之前收集的 IDAT 数据
			// 为收集的 IDAT 数据创建 Reader
			allIDATDataReader := bytes.NewReader(allIDAT)
			// 使用 flate 解压缩收集的 IDAT 数据
			// 注意：由于这里的数据不具备 zlib 头，所以使用 zlib 解压缩会报错
			// 这里采用 flate 进行解压缩
			allIDATReader := flate.NewReader(allIDATDataReader)
			// 创建解压缩缓冲区
			decompressBuf := bytes.Buffer{}
			// 解压缩收集的 IDAT 数据
			_, err := io.Copy(&decompressBuf, allIDATReader)
			// 关闭 Reader
			_ = allIDATReader.Close()
			if err != nil {
				return nil, err
			}
			// 解压缩后的数据放入数组
			decompressedData := decompressBuf.Bytes()

			// 重新编排解压缩后的数据
			var newData []byte
			// 遍历图片的 高
			for y := 0; y < int(imgSize.Height); y++ {
				i := len(newData)
				newData = append(newData, decompressedData[i])
				// 遍历图片的 宽
				for x := 0; x < int(imgSize.Width); x++ {
					i = len(newData)
					newData = append(newData, decompressedData[i+2])
					newData = append(newData, decompressedData[i+1])
					newData = append(newData, decompressedData[i+0])
					newData = append(newData, decompressedData[i+3])
				}
			}

			// 准备压缩重新编排后的数据
			// 创建压缩缓冲区
			var compressBuf bytes.Buffer
			// 注意：虽然解压时使用 flate，但这里解压缩不能使用 flate
			// 需要使用 zlib 进行压缩
			// 创建 zlib 压缩 Writer
			zlibWriter := zlib.NewWriter(&compressBuf)
			// zlib 压缩数据
			_, err = zlibWriter.Write(newData)
			if err != nil {
				return nil, err
			}
			_ = zlibWriter.Close()

			// 压缩后的数据放入数组
			compressedData := compressBuf.Bytes()
			// 压缩后数据的长度
			compressedLen := len(compressedData)

			// 压缩后数据长度写入写入 数据块 头部
			chunkCopy.header.ChunkLength = uint32(compressedLen)
			// 数据块类型改为 IDAT
			copy(chunkCopy.header.ChunkType[:], []byte("IDAT"))
			// 将 数据块 的 图像数据指定为 压缩后的数据
			chunkCopy.data = compressedData

			// 计算 数据块 类型的 crc
			chunkTypeCRC := crc32.Checksum(chunkCopy.header.ChunkType[:], crc32.IEEETable)
			// 基于 数据块 类型的 crc，计算 图像数据 crc
			chunkCopy.crc = crc32.Update(chunkTypeCRC, crc32.IEEETable, chunkCopy.data)
		}

		if !bSkip {
			imgChunks = append(imgChunks, chunkCopy)
		}
	}

	// 生成新的 PngImage
	retPngImg := new(PNGImage)
	retPngImg.imageChunks = imgChunks

	return retPngImg, nil
}

func (receiver *PNGImage) GetImageChunks() []*ImageChunk {
	return receiver.imageChunks
}

// SaveToData 将 PNGImage 中的内容保存到数组
func (receiver *PNGImage) SaveToData() []byte {
	retData := []byte(PngFileHeader)
	for _, imgChunk := range receiver.imageChunks {
		chunkData := imgChunk.SaveFullChunkToData()
		retData = append(retData, chunkData...)
	}
	return retData
}
