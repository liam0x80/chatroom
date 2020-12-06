package server

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
)

type imageType int32

const (
	UNKNOWN imageType = 0
	PNG     imageType = 1
	JPEG    imageType = 2
)

const dirPath = "./image_dir/"

// 下载图片
func imageGetHandleFunc(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		return
	}
	// 组合文件完整路径
	fmt.Println(req.URL.Path)
	fileName := strings.TrimPrefix(req.URL.Path, "/image")
	fmt.Printf("handleGet called, fileName: %s.\n", fileName)
	file, err := os.Open(path.Join(dirPath, fileName))
	defer file.Close()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Println("file, err = os.Open(path)", err)
		return
	}
	w.WriteHeader(http.StatusOK)
	io.Copy(w, file)
}

// 上传图片
func imagePostHandleFunc(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" && req.Method != "PUT" {
		return
	}
	fmt.Println("handlePost called")
	req.ParseMultipartForm(32 << 20)
	data, info, err := req.FormFile("image")
	defer data.Close()
	if err != nil {
		fmt.Println(err)
		return
	}
	imageType := UNKNOWN
	suffix := ""
	// 用buffer先把content-type读进来，然后再把文件指针重置
	// 这里要注意，每次使用buffer，文件指针都会移动
	buff := make([]byte, 512)
	_, err = data.Read(buff)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "read file error.\n")
		return
	}
	fileType := http.DetectContentType(buff)
	fmt.Println(fileType)

	switch http.DetectContentType(buff) {
	case "image/png":
		imageType = PNG
		suffix = ".png"
	case "image/jpg", "image/jpeg":
		imageType = JPEG
		suffix = ".jpg"
	}

	if imageType == UNKNOWN {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "unsupported type received.\n")
		return
	}

	if _, err = data.Seek(0, 0); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "reset error.\n")
		return
	}

	fmt.Printf("info filetype %d, name %s.\n", imageType, info.Filename)
	// 初始化 MD5 实例
	md5Hash := md5.New()
	_, err = io.Copy(md5Hash, data)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "error %s.\n", err.Error())
	}
	// 进行MD5计算，返回16进制的 byte 数组
	fileMd5Fx := md5Hash.Sum(nil)
	fileMd5 := fmt.Sprintf("%x", fileMd5Fx)
	fmt.Printf("fileMd5 is %s.\n", fileMd5)

	// 获取目录信息，并创建目录
	err = os.MkdirAll(dirPath, 0777)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "error %s.\n", err.Error())
		return
	}
	fileName := fileMd5 + suffix
	filePath := path.Join(dirPath, fileName)
	fmt.Printf("filePath is %s.\n", filePath)

	// 存入文件
	if _, err = data.Seek(0, 0); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "data seek error %s.\n", err.Error())
		return
	}

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "error %s.\n", err.Error())
		return
	}
	defer file.Close()
	bytesWritten, err := io.Copy(file, data)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "error %s.", "unknown image type, support type/png, type/jpeg")
	} else {
		fmt.Printf("write %d bytes.\n", bytesWritten)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%s", fileName)
	}
}
