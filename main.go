//go:generate go get -v github.com/josephspurrier/goversioninfo/...
//go:generate goversioninfo -icon=res/app-portable.ico
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/op/go-logging"
)

const (
	NAME            = "slack-portable"
	APP_NAME        = "Slack"
	APP_DATA_FOLDER = "slack"
	APP_PROCESS     = "slack.exe"
)

var (
	log       = logging.MustGetLogger(NAME)
	logFormat = logging.MustStringFormatter(`%{time:2006-01-02 15:04:05} %{level:.4s} - %{message}`)
)

func main() {
	// Current path
	currentPath, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Error("Current path:", err)
	}

	// Logs folder
	var logsPath = path.Join(currentPath, "logs")
	if _, err := os.Stat(logsPath); os.IsNotExist(err) {
		log.Info("Create logs folder", logsPath)
		err = os.Mkdir(logsPath, 777)
		if err != nil {
			log.Error("Create logs folder:", err)
		}
	}

	// Log file
	logfile, err := os.OpenFile(path.Join(logsPath, NAME+".log"), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Error("Log file:", err)
	}

	// Init logger
	logBackendStdout := logging.NewBackendFormatter(logging.NewLogBackend(os.Stdout, "", 0), logFormat)
	logBackendFile := logging.NewBackendFormatter(logging.NewLogBackend(logfile, "", 0), logFormat)
	logging.SetBackend(logBackendStdout, logBackendFile)
	log.Info("--------")
	log.Info("Starting " + NAME + "...")
	log.Info("Current path:", currentPath)

	// Purge logs
	logsFolder, err := os.Open(logsPath)
	if err != nil {
		log.Error("Open logs folder:", err)
	}
	defer logsFolder.Close()
	logsFiles, err := logsFolder.Readdir(-1)
	if err != nil {
		log.Error("Read logs folder:", err)
	}
	log.Info("Reading", logsPath)
	for _, logsFile := range logsFiles {
		if !strings.HasPrefix(logsFile.Name(), NAME) {
			os.Remove(path.Join(logsPath, logsFile.Name()))
			log.Info("Deleted", path.Join(logsPath, logsFile.Name()))
		}
	}

	// Find app folder
	log.Info("Lookup app folder in", currentPath)
	var appPath = ""
	rootFiles, _ := ioutil.ReadDir(currentPath)
	for _, f := range rootFiles {
		if strings.HasPrefix(f.Name(), "app-") && f.IsDir() {
			log.Info("App folder found:", f.Name())
			appPath = path.Join(currentPath, f.Name())
			break
		}
	}
	if _, err := os.Stat(appPath); err == nil {
		log.Info("App path:", appPath)
	} else {
		log.Error("App path does not exist")
	}

	// Init vars
	appExe := path.Join(appPath, APP_PROCESS)
	dataPath := path.Join(currentPath, "data")
	downloadsPath := path.Join(currentPath, "downloads")
	symlinkPath := path.Clean(path.Join(os.Getenv("APPDATA"), APP_DATA_FOLDER))
	slackSettingsPath := path.Join(dataPath, "storage", "slack-settings")
	log.Info("App executable:", appExe)
	log.Info("Data path:", dataPath)
	log.Info("Downloads path:", downloadsPath)
	log.Info("Symlink path:", symlinkPath)
	log.Info("Slack settings path:", slackSettingsPath)

	// Create data folder
	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		log.Info("Create data folder...", dataPath)
		err = os.Mkdir(dataPath, 777)
		if err != nil {
			log.Error("Create data folder:", err)
		}
	}

	// Check old data folder
	if _, err := os.Stat(symlinkPath); err == nil {
		fi, err := os.Lstat(symlinkPath)
		if err != nil {
			log.Error("Symlink lstat:", err)
		}
		if fi.Mode()&os.ModeSymlink != os.ModeSymlink {
			// Copy old data folder
			log.Info("Copy old data from", symlinkPath)
			err = copyDir(symlinkPath, dataPath)
			if err != nil {
				log.Error("Copying old data folder:", err)
			}

			// Rename old data folder
			log.Info("Chmod old data folder...")
			err = os.Chmod(symlinkPath, 0777)
			if err != nil {
				log.Error("Chmod old data folder:", err)
			}

			log.Info("Rename old data folder to", symlinkPath+"_old")
			err = os.Rename(symlinkPath, symlinkPath+"_old")
			if err != nil {
				log.Error("Renaming old data folder:", err)
			}
		}
	}

	// Create symlink
	log.Info("Create symlink", symlinkPath)
	os.Remove(symlinkPath)
	cmd := exec.Command("cmd", "/c", "mklink", "/J", strings.Replace(symlinkPath, "/", "\\", -1), strings.Replace(dataPath, "/", "\\", -1))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Run(); err != nil {
		log.Error("Symlink:", err)
	}
	/*err = os.Symlink(dataPath, symlinkPath)
	  if err != nil {
	    log.Error(err)
	  }*/

	// Check downloads folder
	log.Info("Check downloads folder...")
	if _, err := os.Stat(downloadsPath); os.IsNotExist(err) {
		log.Info("Create download folder", downloadsPath)
		err = os.Mkdir(downloadsPath, 777)
		if err != nil {
			log.Error("Create download folder:", err)
		}
	}

	// Change slack settings
	log.Info("Update Slack settings...")
	if _, err := os.Stat(slackSettingsPath); err == nil {
		rawSettings, err := ioutil.ReadFile(slackSettingsPath)
		if err == nil {
			jsonMapSettings := make(map[string]interface{})
			json.Unmarshal(rawSettings, &jsonMapSettings)
			log.Info("Current settings:", jsonMapSettings)

			jsonMapSettings["resourcePath"] = strings.Replace(appPath+"/resources/app.asar", "/", "\\", -1)
			jsonMapSettings["PrefSSBFileDownloadPath"] = strings.Replace(downloadsPath, "/", "\\", -1)
			log.Info("New settings:", jsonMapSettings)

			jsonSettings, err := json.Marshal(jsonMapSettings)
			if err != nil {
				log.Error("Slack settings marshal:", err)
			}
			err = ioutil.WriteFile(slackSettingsPath, jsonSettings, 0644)
			if err != nil {
				log.Error("Write Slack settings:", err)
			}
		}
	} else {
		log.Warning("Slack settings not found in:", slackSettingsPath)
	}

	// Launch
	log.Infof("Launch %s...", APP_NAME)
	cmd = exec.Command(appExe, "--log-file", "./")
	cmd.Dir = logsPath

	defer logfile.Close()
	cmd.Stdout = logfile
	cmd.Stderr = logfile

	if err := cmd.Start(); err != nil {
		log.Error("Cmd Start:", err)
	}

	cmd.Wait()
}

// src : https://gist.github.com/crazy-max/e50ee72138bb184baf8d1b6e81983f13
func copyDir(src string, dst string) (err error) {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	si, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !si.IsDir() {
		return fmt.Errorf("src is not a directory: %s", src)
	}

	_, err = os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return
	}

	err = os.MkdirAll(dst, si.Mode())
	if err != nil {
		return
	}

	entries, err := ioutil.ReadDir(src)
	if err != nil {
		return
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			err = copyDir(srcPath, dstPath)
			if err != nil {
				return
			}
		} else {
			err = copyFile(srcPath, dstPath)
			if err != nil {
				return
			}
		}
	}

	return
}

func copyFile(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		if e := out.Close(); e != nil {
			err = e
		}
	}()

	_, err = io.Copy(out, in)
	if err != nil {
		return
	}

	err = out.Sync()
	if err != nil {
		return
	}

	si, err := os.Stat(src)
	if err != nil {
		return
	}

	err = os.Chmod(dst, si.Mode())
	if err != nil {
		return
	}

	return
}
