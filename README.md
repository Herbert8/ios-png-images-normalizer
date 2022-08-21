# iOS PNG Images Normalizer




### 一、背景

iOS 打包后的 App 中的 PNG 图像与标准 PNG 图像格式不同，导致在非苹果系统中无法查看。

本工具用于将非标准化的 PNG 图像进行标准化。



### 二、文件格式

1、PNG 文件头

```
\x89PNG\r\n\x1a\n
```

2、多个 PNG 数据块

PNG 数据块结构，分四部分：

（1）0-3 字节：该数据块中，数据部分的长度

（2）4-7 字节：该数据块的块类型，如：CgBI、IHDR、iTXt、tEXt、iDOT、IDAT、IDAT、IEND 等

其中：

- IHDR 块 包含 图片的 宽、高
- IDAT 中包含需要重新编码的数据

（3）数据块中的实际有效数据，长度为 0-3 字节的 32 位证书表示的长度
（4）数据的 CRC32 值



### 三、思路

PNG 图像的格式，在文件头之后，是多个类型的数据块。需要对 IDAT 数据块进行处理：

1、将每个 IDAT 数据块中的 有效数据（不包含块的头信息）进行合并

2、对合并后的数据进行 zlib 解码

3、对解码后的数据重新进行编排

4、对编排后的数据进行 zlib 编码

5、为重新编码后的数据重新生成 IDAT 数据块



### 四、参考资料

- PNG File Format

https://docs.fileformat.com/image/png/

- 隐写术之图片隐写

https://zhuanlan.zhihu.com/p/62895080

- iPhone PNG Images Normalizer

https://axelbrz.com/?mod=iphone-png-images-normalizer

- iPhone PNG Images Normalizer with a fix for multiple IDAT

https://gist.github.com/urielka/3609051