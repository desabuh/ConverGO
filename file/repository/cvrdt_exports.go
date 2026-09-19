package repository

import (
	"fmt"

	"github.com/desabuh/convergo/cvrdt"
	"github.com/desabuh/convergo/file"
	"github.com/desabuh/convergo/history"
	"github.com/desabuh/convergo/utils"
)

//This file defines integrations to adapt a repository for the crdt usage

// a CvrdtRepository is a CRDT state based repository, extension of general Repository
type CvrdtRepository struct {
	*Repository[cvrdt.CvRDTState]
}

func (c *CvrdtRepository) InsertCRDTOpInRepo(op TextualFileOperation) (FileOperationStatus, error) {

	var opType cvrdt.OpType
	if op.OpType == "insert" {
		opType = cvrdt.Insertion
	} else if op.OpType == "delete" {
		opType = cvrdt.Deletion
	} else {
		return FileOperationStatus{}, fmt.Errorf("Supported operation modes are 'insert' or 'delete' not %s", op.OpType)
	}

	newState := cvrdt.GetNewStateFromOp(
		cvrdt.CreateNewLocalOp(opType, op.Pos, op.Content, op.SourceId),
	)

	newInfo := CvrdtFileInfo{
		LocalPath:     op.FilePath,
		ReadOnlyState: newState,
	}

	updatedInfo, err := c.UpdateFileCtx(newInfo)

	if err != nil {
		return FileOperationStatus{}, fmt.Errorf("error while trying to merge new cvrdt operation on %s context: %v", op.FilePath, err)
	}

	//new operations will always be appended at end
	resultingOp := updatedInfo.ReadOnlyState[len(updatedInfo.ReadOnlyState)-1]

	//in deletion when don't know at first what Character was targeted (so we should check last operation in state after the operation)
	return FileOperationStatus{
		Id:         resultingOp.OpId().String(),
		TargetChar: resultingOp.Char(),
		NewFile:    updatedInfo.IsNewlyCreated,
	}, nil

}

type CvrdtRepoFiles = []CvrdtFileInfo

type CvrdtFileInfo = file.FileContextInfo[cvrdt.CvRDTState]

func GetNewCvrdtRepository(siteId string, prefix string, factory func() utils.ObservableState[cvrdt.CvRDTState, string], adapter RepoFilesAdapter[cvrdt.CvRDTState]) *CvrdtRepository {

	return &CvrdtRepository{
		Repository: &Repository[cvrdt.CvRDTState]{
			history:          &history.CvrdtDAGHistory{},
			FileRegistry:     *file.CreateNewCRDTFileRegistry(prefix, factory),
			RepoFilesAdapter: adapter,
		},
	}
}

// this struct is needed to adapt a decoding CRDT implementation C to the general cvrdt.CvrdtState without altering the enclosing FileContext
type CvrdtToRepoFilesAdapter[C cvrdt.CRDTOperation] struct{}

func (crfd CvrdtToRepoFilesAdapter[C]) DecodeToRepoFiles(decoder interface{ Decode(any) error }) (CvrdtRepoFiles, error) {

	var res []file.FileContextInfo[[]C]

	err := decoder.Decode(&res)

	if err != nil {
		return nil, err
	}

	return crfd.getCovarianceCvrdtType(res), nil
}

func (crfd CvrdtToRepoFilesAdapter[C]) getCovarianceCvrdtType(files []file.FileContextInfo[[]C]) CvrdtRepoFiles {
	return utils.Map(files, func(x file.FileContextInfo[[]C]) CvrdtFileInfo {
		result := make(cvrdt.CvRDTState, len(x.ReadOnlyState))

		for i, op := range x.ReadOnlyState {
			result[i] = op
		}

		return CvrdtFileInfo{
			LocalPath:     x.LocalPath,
			ReadOnlyState: result,
		}
	})
}
