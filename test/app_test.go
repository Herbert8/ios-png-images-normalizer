package test

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"os"
	"strings"
	"testing"
	"unsafe"
)

func setup() {
	log.Println("Before all tests ================>>")
}

func teardown() {
	log.Println("<<================= After all tests")
}

func TestApp(t *testing.T) {
	t.Log("Running tests.")

}

//func TestSlice(t *testing.T) {
//	s1 := []byte{1, 2, 3, 4, 5}
//	var s2 []byte
//	s2 = append(nil, s1[2:4]...)
//	s2[0] = 100
//	s2[1] = 200
//	t.Log(s2)
//	t.Log(s1)
//}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}

func TestReader(t *testing.T) {
	s := "abc123\n"
	r := strings.NewReader(s)
	//var bs []byte
	bs := make([]byte, 10)
	bf := bytes.NewBuffer(bs)
	n, err := io.Copy(bf, r)
	//if _, err = io.Copy(os.Stdout, r); err != nil {
	//	t.Error(err.Error())
	//}
	if err != nil {
		t.Error(err.Error())
	}
	t.Log(n)
	t.Log(bs)
	t.Log(string(bs))
}

func bytesToInt(bytesData []byte) int {
	var retUint16 uint16
	byteBuf := bytes.NewBuffer(bytesData)
	_ = binary.Read(byteBuf, binary.LittleEndian, &retUint16)
	return int(retUint16)
}

func TestBin(t *testing.T) {
	bs := []byte{1, 2}
	n := bytesToInt(bs)
	t.Log(n)

}

type MyData struct {
	N1 int16
	N2 int16
}

func TestData(t *testing.T) {
	bs := []byte{1, 2, 2, 1}
	data := MyData{}
	_ = binary.Read(bytes.NewReader(bs), binary.BigEndian, &data)
	t.Log(data.N1)
	t.Log(data.N2)
}

func TestGetSize(t *testing.T) {
	var n int32
	s := unsafe.Sizeof(n)
	t.Log(s)
}

func TestReadReader(t *testing.T) {
	r := strings.NewReader("abcdefg")

	b := make([]byte, 3)

	io.ReadAtLeast(r, b, 2)
	t.Log(b)

	io.ReadAtLeast(r, b, 2)
	t.Log(b)
}

func TestP(t *testing.T) {
	s := "0123456789"
	bs := []byte(s)
	var bs1 []byte
	bs1 = append(bs1, bs[2:8]...)
	bs2 := bs[5:]
	bs1[4] = 1
	t.Log(bs1)
	t.Log(bs2)

	bbb := append([]byte(nil), bs[2:8]...)
	t.Log(bbb)
}
