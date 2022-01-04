package main

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func unzip() {
	dst := "ngrok/"
	archive, err := zip.OpenReader("ngrok.zip")
	if err != nil {
		panic(err)
	}
	defer archive.Close()

	for _, f := range archive.File {
		filePath := filepath.Join(dst, f.Name)
		fmt.Println("unzipping file ", filePath)

		if !strings.HasPrefix(filePath, filepath.Clean(dst)+string(os.PathSeparator)) {
			fmt.Println("invalid file path")
			return
		}
		if f.FileInfo().IsDir() {
			fmt.Println("creating directory...")
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			panic(err)
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			panic(err)
		}

		fileInArchive, err := f.Open()
		if err != nil {
			panic(err)
		}

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			panic(err)
		}

		dstFile.Close()
		fileInArchive.Close()
	}
}

func main() {

	fmt.Println("Opening ngrok site...")

	cmd := exec.Command("powershell", "-nologo", "-noprofile", "Start-Process", "https://ngrok.com")
	err := cmd.Start()

	if err != nil {
		log.Fatal(err)
	}

	var token string

	fmt.Println("Copy the authtoken into the terminal..")
	fmt.Scanln(&token)

	fmt.Println("Downloading ngrok.exe...")
	cmd = exec.Command("powershell", "-nologo", "-noprofile", "Invoke-WebRequest", "https://bin.equinox.io/c/4VmDzA7iaHb/ngrok-stable-windows-amd64.zip", "-OutFile", "ngrok.zip")
	err = cmd.Run()

	if err != nil {
		log.Fatal(err)
	}

	unzip()

	cmd = exec.Command("powershell", "-nologo", "-noprofile", "./ngrok/ngrok.exe", "authtoken", token)
	err = cmd.Run()

	if err != nil {
		log.Fatal(err)
	}

	cmd = exec.Command("powershell", "-nologo", "-noprofile", "Move-Item", "./ngrok/ngrok.exe", "./ngrok.exe")
	err = cmd.Run()

	if err != nil {
		log.Fatal(err)
	}

	cmd = exec.Command("powershell", "-nologo", "-noprofile", "Remove-Item", "-LiteralPath", "./ngrok", "-Force", "-Recurse")
	err = cmd.Run()

	if err != nil {
		log.Fatal(err)
	}

	cmd = exec.Command("powershell", "-nologo", "-noprofile", "Remove-Item", "-LiteralPath", "./ngrok.zip")
	err = cmd.Run()

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Done")

}
