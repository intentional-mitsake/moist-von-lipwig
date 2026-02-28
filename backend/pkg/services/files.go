package services

import (
	"io"
	lg "moist-von-lipwig/pkg/log"
	"net/http"
	"os"
)

var logger = lg.CreateLogger()

func HandleFiles(fieldName string, r *http.Request) ([]string, error) {
	uploadsPath := []string{}
	//get the file/img from the request
	// user can upload multiple files so we are using a slice
	files := r.MultipartForm.File[fieldName] //file is a ptr to multiple fileheader structs
	//this loop runs for every file user has uploaded
	for _, file := range files { //for indx, file[i] := range file(filheaderstructs)
		//create a file to write the file to
		src, err := file.Open() //with f we can do: f.Filename, f.Size, f.Header(for img), f.Filename
		if err != nil {
			logger.Error("Failed to create file", "error", err)
			return nil, err
		}
		defer src.Close()
		//create a path to write the file to
		filePath := "./uploads/" + file.Filename //looks outside the parent dir and for a dir named 'uploads'
		//fmt.Println(filePath)
		dst, err := os.Create(filePath)
		if err != nil {
			logger.Error("Failed to upload the fiel", "error", err)
			return nil, err
		}
		defer dst.Close()
		//copy the file to the path
		if _, err := io.Copy(dst, src); err != nil {
			logger.Error("Failed to copy file", "error", err)
			return nil, err
		}
		//holds path for where all the files have been uploaded
		uploadsPath = append(uploadsPath, filePath)
	}
	return uploadsPath, nil
}
