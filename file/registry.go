package file

import (
	"fmt"
	"os"
	"time"

	"github.com/desabuh/convergo/cvrdt"
	"github.com/desabuh/convergo/utils"
)

type FileRegistry[M utils.Clonable[M]] struct {
	domain              string
	siteId              string
	localDomainPrefix   string
	newEmptyCtxSupplier func() utils.ObservableState[M, string]
	fileContexts        map[string]*StringFileContext[M]
}

func CreateNewWootFileRegistry(siteId string, domain string, domainLocalPrefix string) *FileRegistry[cvrdt.CvRDTState] {
	return CreateNewCRDTFileRegistry(
		siteId,
		domain,
		domainLocalPrefix,
		func() utils.ObservableState[cvrdt.CvRDTState, string] {
			return cvrdt.NewWootCvrdtWithView(siteId)
		},
	)
}

func CreateNewCRDTFileRegistry(siteId string, domain string, domainLocalPrefix string, newCtxSupplier func() utils.ObservableState[cvrdt.CvRDTState, string]) *FileRegistry[cvrdt.CvRDTState] {
	return &FileRegistry[cvrdt.CvRDTState]{
		domain:              domain,
		siteId:              siteId,
		localDomainPrefix:   domainLocalPrefix,
		newEmptyCtxSupplier: newCtxSupplier,
		fileContexts:        make(map[string]*StringFileContext[cvrdt.CvRDTState]),
	}
}

func (fr *FileRegistry[M]) CreateFileCtx(fileName string, pollingInterval time.Duration) error {

	fullPath := fr.localDomainPrefix + fileName

	err := os.MkdirAll(fr.localDomainPrefix, 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", fr.localDomainPrefix, err)
	}

	err = CreateFile(fullPath)

	if err != nil {
		return err
	}

	file, err := OpenFile(fullPath, RW_TRUNC_MODE)

	if err != nil {
		return err
	}

	newCtx := GetNewFileContext(fr.siteId, fr.domain, fileName, fr.newEmptyCtxSupplier(), file, pollingInterval)

	newCtx.TrackState()

	fr.fileContexts[fileName] = newCtx

	return nil

}

func (fr *FileRegistry[M]) UpdateFileCtx(fileCtx FileContextInfo[M]) error {
	ctx, err := fr.getFileCtx(fileCtx.localPath)

	if err != nil {
		return err
	}

	err = ctx.UpdateState(fileCtx.readOnlyState)

	return err

}

func (fr *FileRegistry[M]) GetFileCtxInfo(fileName string) (FileContextInfo[M], error) { // fornisce i dati su un singolo file context readonly (per leggerli)
	ctx, err := fr.getFileCtx(fileName)
	if err != nil {
		return FileContextInfo[M]{}, fmt.Errorf("file context not found")
	}

	return ctx.GetStateCopy(), nil
}

func (fr *FileRegistry[M]) RemoveFileCtx(filename string) error {
	ctx, err := fr.getFileCtx(filename)

	if err != nil {
		return nil
	}

	ctx.UntrackState()

	delete(fr.fileContexts, filename)

	return nil

}

func (fr *FileRegistry[M]) GetAllFileCtxInfo() map[string]FileContextInfo[M] { // fornisce i dati su tutti i file context readonly (per spedirli)
	result := make(map[string]FileContextInfo[M])
	for fileName, ctx := range fr.fileContexts {
		result[fileName] = ctx.GetStateCopy()
	}
	return result
}

func (fr *FileRegistry[M]) getFileCtx(fileName string) (*StringFileContext[M], error) {
	if ctx, exists := fr.fileContexts[fileName]; exists {
		return ctx, nil
	}
	return &StringFileContext[M]{}, fmt.Errorf("file context not found")
}
