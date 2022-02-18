package toolbox

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
)

var dirMode os.FileMode = 0744

// RemoveFileIfExist remove file if exists
func RemoveFileIfExist(filenames ...string) error {
	for _, filename := range filenames {
		if !FileExists(filename) {
			continue
		}
		err := os.Remove(filename)
		if err != nil {
			return err
		}
	}
	return nil
}

// FileExists checks if file exists
func FileExists(filename string) bool {
	if _, err := os.Stat(filename); err != nil {
		return false
	}
	return true
}

// IsDirectory checks if file is directory
func IsDirectory(location string) bool {
	if stat, _ := os.Stat(location); stat != nil {
		return stat.IsDir()
	}
	return false
}

// CreateDirIfNotExist creates directory if they do not exist
func CreateDirIfNotExist(dirs ...string) error {
	for _, dir := range dirs {
		if len(dir) > 1 && strings.HasSuffix(dir, "/") {
			dir = dir[:len(dir)-1]
		}
		parent, _ := path.Split(dir)
		if parent != "/" && parent != dir {
			CreateDirIfNotExist(parent)
		}
		if !FileExists(dir) {
			err := os.Mkdir(dir, dirMode)
			if err != nil {
				return fmt.Errorf("failed to create dir %v %v", dir, err)
			}
		}
	}
	return nil
}

func IsWindows() bool {
	return runtime.GOOS == "windows"
}

func IsMacOs() bool {
	return runtime.GOOS == "darwin"
}

func IsLinux() bool {
	return runtime.GOOS == "linux"
}

func GetAccesibleWindowsDriveString() (driveLetter string, err error) {
	if !IsWindows() {
		err = fmt.Errorf("Runtime OS is not Windows")
		return
	}
	for _, drive := range "CDEFABGHIJKLMNOPQRSTUVWXYZ" {
		f, _err := os.Open(string(drive) + ":\\")
		if _err == nil {
			f.Close()
			driveLetter = string(drive)
		}
	}
	err = fmt.Errorf("There is no accesible drives")
	return
}

/*
func GetOsBasedPath(path string) (osBasedPath string, err error) {
	osBasedPath=path;
	if Ex
	if IsWindows(){
		if strings.HasPrefix(osBasedPath,"/") {
			osBasedPath:=osBasedPath[2:len(osBasedPath)-1]
		}
		if driveLetter, _err := getAccesibleWindowsDriveString(); _err != nil {
			osBasedPath:= driveLetter + ":\\" + strings.Replace(osBasedPath, "/", "\\", -1)
			return
		} else{

		}
	} else {

	}
	if driveLetter, err := getAccesibleWindowsDriveString(); err != nil {
		return driveLetter + ":\\" + strings.Replace(path, "/", "\\", -1)
	} else {
		if strings.HasPrefix(path, "/") {
			return path
		} else {
			return "/" + path
		}
	}
}*/
