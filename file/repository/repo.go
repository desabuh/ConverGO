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
