package main

import (
	"fmt"
	"github.com/Herbert8/ios-png-images-normalizer/pkg/iospng"
	"log"
	"os"
)

const LogoText = `-----------------------------------
     iOS PNG Images Normalizer      
-----------------------------------`

const UsageText = `USAGE:
   ios-png-normalizer <original_png_image> <fixed_png_image>`

func main() {

	fmt.Println(LogoText)

	if nArgCount := len(os.Args); nArgCount < 3 {
		fmt.Println(UsageText)
		os.Exit(1)
	}

	// 通过命令行参数接受原始文件和目标文件
	sOriginalPngFile := os.Args[1]
	sFixedPngFile := os.Args[2]

	// 判断目标文件是否已经存在，已经存在则退出
	bExists, _ := pathExists(sFixedPngFile)
	if bExists {
		fmt.Printf("Target file '%s' already exists.", sFixedPngFile)
		os.Exit(1)
	}

	originalPngFile, err := iospng.ParsePngFile(sOriginalPngFile)
	if err != nil {
		log.Fatalln(err.Error())
	}

	fixedPngFile, err := originalPngFile.Normalize()
	if err != nil {
		log.Fatalln(err.Error())
	}

	fixedPngFileData := fixedPngFile.SaveToData()
	err = os.WriteFile(sFixedPngFile, fixedPngFileData, 0644)
	if err != nil {
		log.Fatalln(err.Error())
	}

	fmt.Println("File normalized successfully.")

}

// Golang 判断文件或文件夹是否存在的方法为使用 os.Stat() 函数返回的错误值进行判断：
//
// 1、如果返回的错误为 nil，说明文件或文件夹存在
// 2、如果返回的错误类型使用 os.IsNotExist() 判断为 true，说明文件或文件夹不存在
// 3、如果返回的错误为其它类型，则不确定是否在存在
func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
