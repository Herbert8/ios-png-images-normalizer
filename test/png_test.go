package test

import (
	"bytes"
	"encoding/binary"
	"github.com/Herbert8/ios-png-images-normalizer/pkg/iospng"
	"os"
	"testing"
)

const pngFile = "/Volumes/data/tmp/bydate/2022-07/2022-07-31/testimg/img/AppIcon60.png"
const newPngFile = "/Volumes/data/tmp/bydate/2022-07/2022-07-31/testimg/img/zzz.png"
const newPngFile1 = "/Volumes/data/tmp/bydate/2022-07/2022-07-31/testimg/img/zzz1.png"

func TestPngHeader(t *testing.T) {
	b, err := iospng.CheckPngFileHeader(pngFile)
	t.Log(err)
	t.Log(b)
}

func TestChunkHeader(t *testing.T) {
	pngData, _ := os.ReadFile(pngFile)
	pngData = pngData[8:]
	pngReader := bytes.NewReader(pngData)
	chunkHeader := iospng.ChunkHeader{}
	_ = binary.Read(pngReader, binary.BigEndian, &chunkHeader)
	t.Log(chunkHeader.ChunkLength)
	t.Log(string(chunkHeader.ChunkType[:]))
}

func TestPngImg(t *testing.T) {
	pngImg, _ := iospng.ParsePngFile(pngFile)
	normalPngImg, _ := pngImg.Normalize()
	data := normalPngImg.SaveToData()
	_ = os.WriteFile(newPngFile1, data, 0644)
}
