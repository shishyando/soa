package better_errors

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func CheckErrorFatal(err error, msg string, v ...any) {
	if CheckError(err, msg, v...) {
		os.Exit(1)
	}
}

func CheckErrorPanic(err error, msg string, v ...any) {
	if CheckError(err, msg, v...) {
		log.Panicln(fmt.Sprintf(msg, v...), err.Error())
	}
}

func CheckError(err error, msg string, v ...any) bool {
	if err != nil {
		log.Println(fmt.Sprintf(msg, v...), err.Error())
		return true
	}
	return false
}

func CheckHttpError(err error, w http.ResponseWriter, code int, msg string, v ...any) bool {
	if err != nil {
		str := fmt.Sprintf(msg, v...)
		log.Println(str, err.Error())
		http.Error(w, str, code)
		return true
	}
	return false
}

func CheckCustom(failed bool, msg string, v ...any) bool {
	if failed {
		log.Printf(msg, v...)
		return true
	}
	return false
}

func CheckCustomFatal(failed bool, msg string, v ...any) {
	if failed {
		os.Exit(1)
	}
}

func CheckCustomHttp(failed bool, w http.ResponseWriter, code int, msg string, v ...any) bool {
	if failed {
		str := fmt.Sprintf(msg, v...)
		log.Println(str)
		http.Error(w, str, code)
		return true
	}
	return false
}
