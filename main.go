package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"unicode"
)

const DbFile = "data.db"

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

var useForwardedHeaders = false
var keyLength = 4

func init() {
	newUseForwardedHeaders, err := strconv.ParseBool(os.Getenv("PASTR_USE_FORWARDED_HEADERS"))
	if err == nil {
		useForwardedHeaders = newUseForwardedHeaders
	} else {
		envValue := os.Getenv("PASTR_USE_FORWARDED_HEADERS")
		if envValue != "" {
			log.Print("Invalid PASTR_USE_FORWARDED_HEADERS: \"", envValue, "\"")
		}
	}
	newKeyLength, err := strconv.Atoi(os.Getenv("PASTR_KEY_LENGTH"))
	if err == nil && newKeyLength >= 4 && newKeyLength <= 12 {
		keyLength = newKeyLength
	} else {
		envValue := os.Getenv("PASTR_KEY_LENGTH")
		if envValue != "" {
			log.Print("Invalid PASTR_KEY_LENGTH: \"", envValue, "\"")
		}
	}
	log.Print("Configuration value PASTR_USE_FORWARDED_HEADERS = ", useForwardedHeaders)
	log.Print("Configuration value PASTR_KEY_LENGTH = ", keyLength)
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			defer r.Body.Close()
			body, _ := io.ReadAll(r.Body)
			key, err := setKey(string(body))
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
				log.Print("Error when posting content: ", err)
			}
			return
		}

		query := r.URL.Path[1:]
		if query == "" {
			http.ServeFile(w, r, "index.html")
			return
		}

		value, err := getKey(query)
		if err == nil && value != "" {
			if isUrl(value) {
				http.Redirect(w, r, value, http.StatusFound)
			}

			fmt.Fprint(w, value)
			return
		}

		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "Not Found")
	})

	http.ListenAndServe(":3000", nil)
}

func getKey(key string) (string, error) {
	if !isAlphanumeric(key) {
		return "", errors.New("key is not alphanumeric")
	}

	file, err := os.Open(DbFile)
	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		text := scanner.Text()
		if strings.HasPrefix(text, key+" ") {
			return text[len(key+" "):], scanner.Err()
		}
	}

	return "", scanner.Err()
}

func setKey(value string) (string, error) {
	file, err := os.OpenFile(DbFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", err
	}
	defer file.Close()

	key := genKey()
	for {
		content, err := getKey(key)
		if err != nil {
			return "", err
		}

		if content == "" {
			break
		}

		key = genKey()
	}
	_, err2 := file.WriteString(key + " " + value + "\n")

	if err2 != nil {
		return "", err2
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
