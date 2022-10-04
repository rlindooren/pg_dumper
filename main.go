// A simple "http-wrapper" around pg_dump.
// Allowing to easily trigger (by invoking an HTTP endpoint) the dumping and restoring of database dumps.
// Source: https://github.com/rlindooren/pg_dumper

package main

import (
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var config *Config

func main() {
	http.HandleFunc("/list", listDumps)
	http.HandleFunc("/dump", createDump)
	http.HandleFunc("/restore", restoreDump)
	http.HandleFunc("/delete", deleteDump)
	http.HandleFunc("/download", downloadDump)
	http.HandleFunc("/health", health)
	config = readConfig()
	addr := fmt.Sprintf("%s:%s", config.host, config.port)
	log.Printf("Starting (binding to '%s', config = %+v) \n", addr, config)
	log.Println("Source: https://github.com/rlindooren/pg_dumper")
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func listDumps(w http.ResponseWriter, _ *http.Request) {
	fileInfos, err := getFileInfoOfExistingDumps()
	if err != nil {
		errMsg := fmt.Sprintf("Error while listing dump files %s", err)
		handleError(w, http.StatusInternalServerError, errMsg, err)
		return
	}
	for _, fileInfo := range fileInfos {
		fileName := fileInfo.Name()
		extIdx := strings.LastIndex(fileName, fmt.Sprintf(".%s", config.dumpfileExt))
		dumpName := fileName[0:extIdx]
		fmt.Fprintf(w, "%s (%s%s @ %s)\n", dumpName, config.dir, fileInfo.Name(), fileInfo.ModTime())
	}
}

func createDump(w http.ResponseWriter, req *http.Request) {
	name, err := getParamValueFromRequestExpectingItToExist("name", req)
	if err != nil {
		errMsg := fmt.Sprintf("%s", err)
		handleError(w, http.StatusBadRequest, errMsg, err)
		return
	}
	fileName := getAbsolutePathForDumpName(name)
	var args []string
	args = append(args, config.dumpArgs...)
	args = append(args, "-f", fileName)
	executeCommand("pg_dump", args, w)
}

func restoreDump(w http.ResponseWriter, req *http.Request) {
	name, err := getParamValueFromRequestExpectingItToExist("name", req)
	if err != nil {
		errMsg := fmt.Sprintf("%s", err)
		handleError(w, http.StatusBadRequest, errMsg, err)
		return
	}

	fileName := getAbsolutePathForDumpName(name)
	var command string
	var args []string

	// When the file has a '.sql' extension, then considering it to be a plain text dump
	sqlFile, _ := regexp.MatchString("^.*\\.sql$", fileName)
	if sqlFile {
		command = "psql"
	} else {
		command = "pg_restore"
		args = append(args, config.restoreArgs...)
	}
	args = append(args, "-f", fileName)
	executeCommand(command, args, w)
}

func deleteDump(w http.ResponseWriter, req *http.Request) {
	name, err := getParamValueFromRequestExpectingItToExist("name", req)
	if err != nil {
		errMsg := fmt.Sprintf("%s", err)
		handleError(w, http.StatusBadRequest, errMsg, err)
		return
	}

	fileName := getAbsolutePathForDumpName(name)
	err = exec.Command("rm", fileName).Run()
	if err != nil {
		errMsg := fmt.Sprintf("Error while deleting dump '%s' %s", fileName, err)
		handleError(w, http.StatusInternalServerError, errMsg, err)
	}
}

func downloadDump(w http.ResponseWriter, req *http.Request) {
	name, err := getParamValueFromRequestExpectingItToExist("name", req)
	if err != nil {
		errMsg := fmt.Sprintf("%s", err)
		handleError(w, http.StatusBadRequest, errMsg, err)
		return
	}

	fileName := getAbsolutePathForDumpName(name)
	fileNameWithoutPath := fileName[strings.LastIndex(fileName, "/")+1:]
	println(fileNameWithoutPath)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", strconv.Quote(fileNameWithoutPath)))
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, req, fileName)
}

func executeCommand(command string, args []string, w http.ResponseWriter) {
	cmd := exec.Command(command, args...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	output, _ := ioutil.ReadAll(stderr)
	fmt.Printf("%s\n", output)
	if err := cmd.Wait(); err != nil {
		errMsg := fmt.Sprintf("Error executing %s with arguments: %s):\n%s", command, args, output)
		handleError(w, http.StatusInternalServerError, errMsg, err)
	} else {
		msg := fmt.Sprintf("Succesfully executed %s with arguments: %s):\n%s", command, args, output)
		handleSuccess(w, msg)
	}
}

func getFileInfoOfExistingDumps() ([]fs.FileInfo, error) {
	files, err := ioutil.ReadDir(config.dir)
	if err != nil {
		return nil, err
	}
	filenamePattern := fmt.Sprintf("^.*\\.%s$", config.dumpfileExt)
	var fileInfos []fs.FileInfo
	for _, file := range files {
		match, _ := regexp.MatchString(filenamePattern, file.Name())
		if match && !file.IsDir() {
			fileInfos = append(fileInfos, file)
		}
	}
	return fileInfos, nil
}

func getAbsolutePathForDumpName(name string) string {
	return fmt.Sprintf("%s%s.%s", config.dir, name, config.dumpfileExt)
}

// Can be used to see if the application responds and can connect to the database (e.g. a health check)
func health(w http.ResponseWriter, _ *http.Request) {
	versionBytes, err := exec.Command("pg_dump", "--version").Output()
	if err != nil {
		errMsg := fmt.Sprintf("Error while getting version of pg_dump %s", err)
		handleError(w, http.StatusInternalServerError, errMsg, err)
		return
	}

	isReadyBytes, err := exec.Command("pg_isready").Output()
	if err != nil {
		errMsg := fmt.Sprintf("Error while executing pg_isready %s", err)
		handleError(w, http.StatusInternalServerError, errMsg, err)
		return
	}

	_, _ = fmt.Fprintln(w, "OK")
	fmt.Fprintf(w, "pg_dump version: %v", string(versionBytes))
	fmt.Fprintf(w, "pg_isready: %v", string(isReadyBytes))
}

type Config struct {
	host        string
	port        string
	dir         string
	dumpfileExt string
	dumpArgs    []string
	restoreArgs []string
}

func readConfig() *Config {
	dumpArgs := strings.Split(getEnvVariableOrDefault("PG_DUMP_DATA_ARGS", "--clean --format=plain"), " ")
	restoreArgs := strings.Split(getEnvVariableOrDefault("PG_RESTORE_ARGS", ""), " ")
	dumpfileExt := determineDumpFileExtension(dumpArgs)
	config := Config{
		host:        getEnvVariableOrDefault("HOST", ""),
		port:        getEnvVariableOrDefault("PORT", "8090"),
		dir:         getEnvVariableOrDefault("DIR", os.TempDir()),
		dumpfileExt: dumpfileExt,
		dumpArgs:    dumpArgs,
		restoreArgs: restoreArgs,
	}
	return &config
}

func getEnvVariableOrDefault(name, defaultVal string) string {
	value := os.Getenv(name)
	if len(value) == 0 {
		log.Printf("No value for environment variable '%s' using default value '%s'\n", name, defaultVal)
		return defaultVal
	}
	return value
}

func getParamValueFromRequestExpectingItToExist(queryParamName string, req *http.Request) (string, error) {
	value := req.URL.Query().Get(queryParamName)
	if len(value) == 0 {
		errMsg := fmt.Sprintf("No value provided for query parameter '%s'", queryParamName)
		return "", errors.New(errMsg)
	}
	return value, nil
}

func handleSuccess(w http.ResponseWriter, optionalMsg string) {
	_, _ = fmt.Fprintln(w, "SUCCESS")
	if len(optionalMsg) != 0 {
		log.Println(optionalMsg)
		_, _ = fmt.Fprintln(w, optionalMsg)
	}
}

func handleError(w http.ResponseWriter, statusCode int, errMsg string, _ error) {
	log.Println(errMsg)
	w.WriteHeader(statusCode)
	_, _ = fmt.Fprintln(w, "ERROR")
	fmt.Fprintf(w, "%s\n", errMsg)
}

func determineDumpFileExtension(dumpArgs []string) string {
	pattern := regexp.MustCompile("^--format=(.*)$")
	var format string
	for _, dumpArg := range dumpArgs {
		if pattern.MatchString(dumpArg) {
			format = pattern.FindStringSubmatch(dumpArg)[1]
			break
		}
	}
	if format == "plain" {
		return "dump.sql"
	} else if format == "custom" {
		return "dump.custom"
	} else if format == "tar" {
		return "dump.tar"
	} else {
		log.Fatal("No output format specified for dumping. Please provide '--format=plain', '--format=custom' or '--format=tar'")
		return "?"
	}
}
