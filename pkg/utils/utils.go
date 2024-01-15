package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/skamensky/email-archiver/pkg/models"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

func getRuntimeCallerInfoAsString() string {
	dirDepthLimit := 2
	_, file, line, ok := runtime.Caller(2)

	if !ok {
		return "no-caller-info"
	}

	directories := strings.Split(filepath.Dir(file), string(os.PathSeparator))

	// traverse directories backwards until we reach the limit
	lastDirectory := ""
	for i := len(directories) - 1; i >= 0 && i >= len(directories)-dirDepthLimit; i-- {
		lastDirectory = filepath.Join(lastDirectory, directories[i])
	}
	// reverse the string
	directories = strings.Split(lastDirectory, string(os.PathSeparator))
	finalDirectory := ""
	for i := len(directories) - 1; i >= 0; i-- {
		finalDirectory = filepath.Join(finalDirectory, directories[i])
	}
	finalFile := filepath.Join(finalDirectory, filepath.Base(file))
	return fmt.Sprintf("%s:%d", finalFile, line)
}

/*
JoinErrors joins an error with a message and returns a new error with formatted file names and line numbers
it also preserves the 'nilness' of the error for easy chaining
*/
func JoinErrors(message string, err error) error {
	if err == nil {
		return nil
	}
	callerInfo := getRuntimeCallerInfoAsString()
	message = fmt.Sprintf("%s %s", callerInfo, message)
	if !strings.HasSuffix(message, " ") {
		message += " "
	}
	return errors.Join(errors.New(message), err)
}

func MustJSON(i interface{}) string {
	if reflect.TypeOf(i).Kind() == reflect.Struct {
		var mapInterface map[string]interface{}
		err := mapstructure.Decode(i, &mapInterface)
		if err != nil {
			log.Fatal("failed to decode interface to map ", err)
		}

		jsonD, err := json.Marshal(mapInterface)
		if err != nil {
			log.Fatal("failed to marshal map to json ", err)
		}
		return string(jsonD)
	} else {
		jsonD, err := json.Marshal(i)
		if err != nil {
			log.Fatal(fmt.Sprintf("failed to marshal %v to json ", i), err)
		}
		return string(jsonD)
	}
}

func IsInterfaceNil(i interface{}) bool {
	return reflect.DeepEqual(i, reflect.Zero(reflect.TypeOf(i)).Interface())
}

type Set[T comparable] map[T]int

func (set Set[T]) Contains(key T) bool {
	_, ok := set[key]
	return ok
}

func (set Set[T]) Add(key T) {
	set[key] = 0
}

func (s Set[T]) Intersection(other Set[T]) Set[T] {
	s3 := make(Set[T])
	for k := range s {
		if other.Contains(k) {
			s3.Add(k)
		}
	}
	return s3
}

func (s Set[T]) Minus(other Set[T]) Set[T] {
	s3 := make(Set[T])
	for k := range s {
		if !other.Contains(k) {
			s3.Add(k)
		}
	}
	return s3
}

func (s Set[T]) ToSlice() []T {
	slice := make([]T, 0, len(s))
	for k := range s {
		slice = append(slice, k)
	}
	return slice
}

func NewSet[T comparable](slice []T) Set[T] {
	m := make(Set[T])
	for _, s := range slice {
		m[s] = 0
	}
	return m
}

func ReverseArray[T any](array []T) []T {
	for i := len(array)/2 - 1; i >= 0; i-- {
		opp := len(array) - 1 - i
		array[i], array[opp] = array[opp], array[i]
	}
	return array
}

func DebugPrintln(message string) {
	// check if DEBUG is in the environment and its true:
	debug, ok := os.LookupEnv(models.DEBUG_ENVIRONMENT_KEY)
	if !ok {
		return
	}

	debuBool, err := strconv.ParseBool(debug)

	if err != nil || !debuBool {
		return
	}

	callerInfo := getRuntimeCallerInfoAsString()
	message = fmt.Sprintf("DEBUG: %s %s", callerInfo, message)
	log.Println(message)
}
