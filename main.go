package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"unicode"
)

const maxUrlLength = 32768 // 16x max length defined in RFC, should be good enough to handle Unicode URLs

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

var dataPath = "data"
var useForwardedHeaders = false
var keyLength = 4

func init() {
	envValue := os.Getenv("PASTR_DATA_PATH")
	if envValue != "" {
		dataPath = envValue
	}
	// Get absolute data path
	newDataPath, err := filepath.Abs(dataPath)
	if err == nil {
		// Path is well-formed, check if it exists
		if _, err := os.Stat(newDataPath); !os.IsNotExist(err) {
			dataPath = newDataPath
		} else {
			// If path does not exist, attempt to create it
			err := os.MkdirAll(newDataPath, os.ModePerm)
			if err == nil {
				dataPath = newDataPath
			} else {
				log.Panic("Invalid data path: \"", newDataPath, "\": ", err)
			}
		}
	} else {
		log.Panic("Invalid data path: \"", newDataPath, "\": ", err)
	}

	envValue2 := os.Getenv("PASTR_USE_FORWARDED_HEADERS")
	if envValue2 != "" {
		newUseForwardedHeaders, err := strconv.ParseBool(envValue2)
		if err == nil {
			useForwardedHeaders = newUseForwardedHeaders
		} else {
			log.Print("Invalid PASTR_USE_FORWARDED_HEADERS: \"", envValue2, "\"")
		}
	}

	envValue3 := os.Getenv("PASTR_KEY_LENGTH")
	if envValue3 != "" {
		newKeyLength, err := strconv.Atoi(envValue3)
		if err == nil && newKeyLength >= 4 && newKeyLength <= 12 {
			keyLength = newKeyLength
		} else {
			log.Print("Invalid PASTR_KEY_LENGTH: \"", envValue3, "\"")
		}
	}

	log.Print("Configuration value PASTR_DATA_PATH = ", dataPath)
	log.Print("Configuration value PASTR_USE_FORWARDED_HEADERS = ", useForwardedHeaders)
	log.Print("Configuration value PASTR_KEY_LENGTH = ", keyLength)
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			defer r.Body.Close()
			body, _ := io.ReadAll(r.Body)
			key, err := setKey(body)
			if err == nil {
				scheme := "http"
				host := r.Host
				if useForwardedHeaders {
					scheme = r.Header.Get("X-Forwarded-Proto")
					host = r.Header.Get("X-Forwarded-Host")
				}
				url := url.URL{
					Scheme: scheme,
					Host:   host,
					Path:   key,
				}
				fmt.Fprint(w, url.String())
				log.Printf("New content posted at key %s", key)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
				log.Print("Error when posting content: ", err)
			}
			return
		}

		query := r.URL.Path[1:]
		if query == "" {
			http.ServeFile(w, r, "index.html")
			return
		}

		value, numberOfReadBytes, err := getKey(query)
		if err == nil && value != nil {
			valueAsString := string(value[:numberOfReadBytes])
			if isUrl(valueAsString) {
				http.Redirect(w, r, valueAsString, http.StatusFound)
				return
			}

			serveKey(query, w, r)
			return
		}

		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "Not Found")
	})

	http.ListenAndServe(":3000", nil)
}

func getKey(key string) ([]byte, int, error) {
	if !isAlphanumeric(key) {
		return nil, 0, errors.New("key is not alphanumeric")
	}

	file, err := os.Open(filepath.Join(dataPath, key))
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	buffer := make([]byte, maxUrlLength)
	numberOfReadBytes, err2 := file.Read(buffer)
	if err2 != nil && err2 != io.EOF {
		return nil, 0, err2
	}

	return buffer, numberOfReadBytes, nil
}

func serveKey(key string, w http.ResponseWriter, r *http.Request) {
	if !isAlphanumeric(key) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	http.ServeFile(w, r, filepath.Join(dataPath, key))
}

func setKey(value []byte) (string, error) {
	// Generate key
	key := genKey()
	for {
		_, _, err := getKey(key)

		if errors.Is(err, os.ErrNotExist) {
			break
		}

		if err != nil {
			return "", err
		}

		key = genKey()
	}

	// Write content to file
	err := os.WriteFile(filepath.Join(dataPath, key), value, 0644)
	if err != nil {
		return "", err
	}

	return key, nil
}

func isUrl(value string) bool {
	_, err := url.ParseRequestURI(value)
	return err == nil
}

func isAlphanumeric(value string) bool {
	for _, c := range value {
		if !unicode.IsDigit(c) && !unicode.IsLetter(c) {
			return false
		}
	}

	return true
}

func genKey() string {
	bytes := make([]rune, keyLength)
	for i := range bytes {
		bytes[i] = letters[rand.Intn(len(letters))]
	}
	return string(bytes)
}
