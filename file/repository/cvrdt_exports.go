package repository

import (
	"github.com/desabuh/convergo/cvrdt"
	"github.com/desabuh/convergo/file"
	"github.com/desabuh/convergo/history"
	"github.com/desabuh/convergo/utils"
)

// a CvrdtRepository is a CRDT state based repository
type CvrdtRepository = Repository[cvrdt.CvRDTState]

type CvrdtRepoFiles = []CvrdtFileInfo

type CvrdtFileInfo = file.FileContextInfo[cvrdt.CvRDTState]

func GetNewCvrdtRepository(siteId string, prefix string, factory func() utils.ObservableState[cvrdt.CvRDTState, string]) *Repository[cvrdt.CvRDTState] {

	return &Repository[cvrdt.CvRDTState]{
		history:      &history.CvrdtDAGHistory{},
		FileRegistry: *file.CreateNewCRDTFileRegistry(prefix, factory),
	}
}

// this struct defines a behavior to adapt a decoding approach to the specific
// RepoFiles by providing a general crdt.CRDTOperation
// (It also needed for golang missing covariance feature for generics)

type CvrdtToRepoFilesAdapter[X interface{ Decode(any) error }, Y cvrdt.CRDTOperation] struct{}

func (crfd *CvrdtToRepoFilesAdapter[X, Y]) DecodeToRepoFiles(decoder X) (CvrdtRepoFiles, error) {

	var res []file.FileContextInfo[[]Y]

	err := decoder.Decode(&res)

	if err != nil {
		return nil, err
	}

	return utils.Map(res, func(x file.FileContextInfo[[]Y]) CvrdtFileInfo {
		result := make(cvrdt.CvRDTState, len(x.ReadOnlyState))

		for i, op := range x.ReadOnlyState {
			result[i] = op
		}

		return CvrdtFileInfo{
			LocalPath:     x.LocalPath,
			ReadOnlyState: result,
		}
	}), nil
}
