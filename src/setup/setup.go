package main

import (
	"fmt"
	"log"
	"os/exec"
)

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
	cmd = exec.Command("powershell", "-nologo", "-noprofile", "Invoke-WebRequest", "https://bin.equinox.io/c/4VmDzA7iaHb/ngrok-stable-windows-amd64.zip", "-OutFile", "ngrok.exe")
	err = cmd.Start()

	if err != nil {
		log.Fatal(err)
	}

	cmd = exec.Command("powershell", "-nologo", "-noprofile", ".\\ngrok.exe", "authtoken", token)
	err = cmd.Start()

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Done")

}
