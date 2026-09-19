package repository

import (
	"errors"
	"fmt"

	"github.com/desabuh/convergo/file"
	"github.com/desabuh/convergo/history"
	"github.com/desabuh/convergo/utils"
)

// A repository is a file registry with additional history management capabilities
type Repository[S utils.Mergeable[S]] struct {
	history history.History[S]
	file.FileRegistry[S]
	RepoFilesAdapter[S]
}

func (r *Repository[S]) DisplayFileHistory(filePath string, mode string, args map[string]any) (string, error) {
	info, err := r.FileRegistry.GetFileCtxInfo(filePath)

	if err != nil {
		return "", fmt.Errorf("No file named %s was found in the repository: %v", filePath, err)
	}

	err = r.history.UpdateHistory(info.ReadOnlyState)

	if err != nil {
		return "", fmt.Errorf("Error while trying to update repository history: %v", err)
	}

	args["mode"] = mode

	return r.history.Display(args)
}

func (r *Repository[S]) GetRepoFiles() []file.FileContextInfo[S] {
	return utils.Values(r.FileRegistry.GetAllFileCtxInfo())
}

func (r *Repository[S]) ShutDownRepo() error {
	var errs []error

	if err := r.FileRegistry.RemoveAllCtx(); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

// This interface is needed to adapt some deserializing approach to return a list of context info
type RepoFilesAdapter[S any] interface {
	DecodeToRepoFiles(decoder interface{ Decode(any) error }) ([]file.FileContextInfo[S], error)
}

// provide info about resulting operation like final id assigned to operation, the char interested to it and wether a new file was created or not
type FileOperationStatus struct {
	Id         string
	TargetChar string
	NewFile    bool
}

// define a general textual operation applied on a repository file
type TextualFileOperation struct {
	SourceId string
	FilePath string
	OpType   string
	Pos      int
	Content  string
}
