package main

import (
	"archive/zip"
	"compress/zlib"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
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
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		fmt.Printf("Unable to read credentials.json file. Err: %v\n", err)
		return nil, err
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, drive.DriveFileScope)

	if err != nil {
		return nil, err
	}

	client := getClient(config)

	service, err := drive.New(client)

	if err != nil {
		fmt.Printf("Cannot create the Google Drive service: %v\n", err)
		return nil, err
	}

	return service, err
}

func createDir(service *drive.Service, name string, parentId string) (*drive.File, error) {
	d := &drive.File{
		Name:     name,
		MimeType: "application/vnd.google-apps.folder",
		Parents:  []string{parentId},
	}

	file, err := service.Files.Create(d).Do()

	if err != nil {
		log.Println("Could not create dir: " + err.Error())
		return nil, err
	}

	return file, nil
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
		fmt.Println(err)
	}

	for _, file := range files {
		fmt.Println(basePath + file.Name())
		if !file.IsDir() {
			dat, err := ioutil.ReadFile(basePath + file.Name())
			if err != nil {
				fmt.Println(err)
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

func fetch(service *drive.Service) {
	r, err := service.Files.List().Q("'-Minecraft-' in parents").OrderBy("createdTime desc").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve files: %v", err)
	}
	fmt.Println("Files:")
	if len(r.Files) == 0 {
		fmt.Println("No files found.")
	} else {
		for _, i := range r.Files {
			fmt.Printf("%v (%vs )\n", i.Name, i.Id)
		}

		f_id := r.Files[0].Id

		response, err := service.Files.Get(f_id).Download()

		if err != nil {
			fmt.Println(err)
		}

		z, errZ := zlib.NewReader(response.Body)

		if errZ != nil {
			fmt.Println(errZ)
		}

		response.Body.Close()

		p, errA := ioutil.ReadAll(z)

		if errA != nil {
			fmt.Println(errA)
		}

		errF := os.WriteFile("server.zip", p, 0644)

		if errF != nil {
			fmt.Println(errF)
		}

	}

}

func main() {

	err := godotenv.Load(".env")

	if err != nil {
		log.Fatal("Error loading .env file")
	}

	SERVER_PATH := os.Getenv("SERVER_PATH")

	// Step 1. Get the Google Drive service
	service, err := getService()

	// Fetch last version
	fetch(service)

	//RUN MINECRAFT && ngrok

	ngrokCmd := exec.Command("ngrok", "tcp", "25565", "-log=stdout")
	ngrokCmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	err = ngrokCmd.Wait()

	log.Printf("Ngrok finished with error: %v", err)

	mcCmd := exec.Command("java", "-Xmx1024M", "-Xms1024M", "-jar", "server.jar", "nogui")
	mcCmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	err = mcCmd.Wait()

	log.Printf("Minecraft finished with error: %v", err)

	//------------------------

	baseFolder := SERVER_PATH

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

	// Step 1. Open the file
	f, err := os.Open("mc-" + id + ".zip")

	if err != nil {
		panic(fmt.Sprintf("cannot open file: %v", err))
	}

	defer f.Close()

	// Step 3. Create the directory
	dir, err := createDir(service, "-Minecraft-", "root")

	if err != nil {
		panic(fmt.Sprintf("Could not create dir: %v\n", err))
	}

	// Step 4. Create the file and upload its content
	file, err := createFile(service, "mc-"+id+".zip", "application/zip", f, dir.Id)

	if err != nil {
		panic(fmt.Sprintf("Could not create file: %v\n", err))
	}

	fmt.Printf("File '%s' successfully uploaded in '%s' directory", file.Name, dir.Name)

}
