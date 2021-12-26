package main

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func getService() (*drive.Service, error) {
	ctx := context.Background()
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, drive.DriveScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Drive client: %v", err)
	}

	return srv, err
}

func getDir(service *drive.Service) (*drive.File, error) {
	folder, err := service.Files.List().Q("name = 'Minecraft' and mimeType = 'application/vnd.google-apps.folder'").Do()

	if err != nil {
		log.Println(err.Error())
		return nil, err
	}

	if len(folder.Files) != 1 {
		return nil, nil
	}

	return folder.Files[0], nil

}

func createFile(service *drive.Service, name string, mimeType string, content io.Reader, parentId string) (*drive.File, error) {
	f := &drive.File{
		MimeType: mimeType,
		Name:     name,
		Parents:  []string{parentId},
	}
	file, err := service.Files.Create(f).Media(content).Do()

	if err != nil {
		log.Println("Could not create file: " + err.Error())
		return nil, err
	}

	return file, nil
}

func addFiles(w *zip.Writer, basePath, baseInZip string) {
	// Open the Directory
	files, err := ioutil.ReadDir(basePath)
	if err != nil {
		log.Println(err)
	}

	for _, file := range files {
		log.Println(basePath + file.Name())
		if !file.IsDir() {
			dat, err := ioutil.ReadFile(basePath + file.Name())
			if err != nil {
				log.Println(err)
			}

			// Add some files to the archive.
			f, err := w.Create(baseInZip + file.Name())
			if err != nil {
				fmt.Println(err)
			}
			_, err = f.Write(dat)
			if err != nil {
				fmt.Println(err)
			}
		} else if file.IsDir() {

			// Recurse
			newBase := basePath + file.Name() + "/"
			fmt.Println("Recursing and Adding SubDir: " + file.Name())
			fmt.Println("Recursing and Adding SubDir: " + newBase)

			addFiles(w, newBase, baseInZip+file.Name()+"/")
		}
	}
}

func unzip() {
	dst := "Minecraft"
	archive, err := zip.OpenReader("server.zip")
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

func fetch(service *drive.Service) {
	d, err := getDir(service)

	if err != nil {
		log.Fatal(err)
	}

	r, err := service.Files.List().Q("parents in " + "'" + d.Id + "'").OrderBy("createdTime desc").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve files: %v", err)
	}
	fmt.Println("Files:")
	if len(r.Files) == 0 {
		fmt.Println("No files found.")
	} else {

		f_id := r.Files[0].Id

		fmt.Println(r.Files[0].Name)

		response, err := service.Files.Get(f_id).Download()

		if err != nil {
			fmt.Println(err)
		}

		bodybytes, err := io.ReadAll(response.Body)

		fmt.Print(len(bodybytes))

		if err != nil {
			log.Fatal(err)
		}

		ioutil.WriteFile("server.zip", bodybytes, 0777)

		unzip()

		for j, i := range r.Files {
			fmt.Printf("%v (%vs )\n", i.Name, i.Id)
			if j != 0 {
				service.Files.Delete(i.Id).Do()
			}
		}

	}

}

func spawn_processes() {
	fmt.Println("START ngrok")
	ngrokCmd := exec.Command("cmd.exe", "/C", "cd", "Minecraft/Minecraft/", "&&", "ngrok.exe", "tcp", "25565", "-log=stdout")

	ngrokCmd.Stdout = os.Stdout
	ngrokCmd.Stderr = os.Stderr

	err := ngrokCmd.Start()

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("END ngrok")
	serverCmd := exec.Command("cmd.exe", "/C", "cd", "Minecraft/Minecraft/", "&&", "java", "-Xmx1024M", "-Xms1024M", "-jar", "server.jar", "nogui")
	stdin, err := serverCmd.StdinPipe()
	if err != nil {
		fmt.Println(err) //replace with logger, or anything you want
	}

	serverCmd.Stdout = os.Stdout
	serverCmd.Stderr = os.Stderr

	fmt.Println("START")
	if err = serverCmd.Start(); err != nil {
		fmt.Println("An error occured: ", err)
	}

	fmt.Scanln()
	fmt.Println("\nCLOSING THE SERVER...")
	fmt.Println()
	fmt.Println()
	io.WriteString(stdin, "/stop\n")
	serverCmd.Wait()
	ngrokCmd.Process.Kill()
	ngrokCmd.Process.Release()
	killcmd := exec.Command("cmd.exe", "/c", "taskkill", "/im", "ngrok.exe", "/t", "/f").Run()
	if killcmd != nil {
		fmt.Println("An error occured: ", killcmd)
	}
	fmt.Println()
	fmt.Println()
	fmt.Println("END")
}

func upload(path string, service *drive.Service) string {
	baseFolder := path

	id := uuid.New().String()

	// Get a Buffer to Write To
	outFile, err := os.Create("mc-" + id + ".zip")
	if err != nil {
		fmt.Println(err)
	}
	defer outFile.Close()

	// Create a new zip archive.
	w := zip.NewWriter(outFile)

	// Add some files to the archive.
	addFiles(w, baseFolder, "")

	if err != nil {
		fmt.Println(err)
	}

	// Make sure to check the error on Close.
	err = w.Close()
	if err != nil {
		fmt.Println(err)
	}

	f, err := os.Open("mc-" + id + ".zip")

	if err != nil {
		log.Fatal(fmt.Sprintf("cannot open file: %v", err))
	}

	defer f.Close()

	dir, err := getDir(service)

	if err != nil {
		log.Fatal(fmt.Sprintf("Could not create dir: %v\n", err))
	}

	file, err := createFile(service, "mc-"+id+".zip", "application/zip", f, dir.Id)

	if err != nil {
		log.Fatal(fmt.Sprintf("Could not create file: %v\n", err))
	}

	fmt.Printf("File '%s' successfully uploaded in '%s' directory", file.Name, dir.Name)

	return id
}

func remove_temp(id string) {
	err := os.RemoveAll("Minecraft/")
	if err != nil {
		fmt.Println(err)
	}

	err = os.Remove("server.zip")
	if err != nil {
		fmt.Println(err)
	}

	err = os.Remove("mc-" + id + ".zip")
	if err != nil {
		fmt.Println(err)
	}
}

func main() {

	service, err := getService()

	if err != nil {
		log.Fatal(err)
	}

	// Fetch last version
	fetch(service)

	//RUN MINECRAFT && ngrok
	spawn_processes()
	//------------------------
	id := upload("Minecraft/", service)
	remove_temp(id)

}
