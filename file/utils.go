package file

import (
	"errors"
	"fmt"
	"os"
)

const RW_TRUNC_MODE = os.O_RDWR | os.O_TRUNC
const FULL_BIT_PERMISSION = 0666

func IsFileExists(path string) (bool, error) {
	_, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func CreateFile(path string) error {
	// Check if file already exists
	exist, err := IsFileExists(path)
	if err != nil {
		return err
	}

	if exist {
		return fmt.Errorf("file already exists: %s", path)
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return nil

}

func OpenFile(path string, access_mode int) (*os.File, error) {
	exist, err := IsFileExists(path)

	if !exist {
		return nil, err
	}

	f, err := os.OpenFile(path, access_mode, FULL_BIT_PERMISSION)
	if err != nil {
		return nil, err
	}

	return f, nil
}
