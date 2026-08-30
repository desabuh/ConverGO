package history

import (
	"fmt"

	"github.com/desabuh/convergo/cvrdt"
	"github.com/desabuh/convergo/utils"
)

type History[X utils.Mergeable[X]] interface {
	UpdateHistory(source X) error
	Display(args map[string]any) (string, error)
}

type CvrdtDAGHistory struct {
	currentDag OperationDAG
}

func (cdh *CvrdtDAGHistory) UpdateHistory(newState cvrdt.CvRDTState) error {
	cdh.currentDag = *BuildDAG(newState)
	return nil
}

func (cdh *CvrdtDAGHistory) Display(args map[string]any) (string, error) {
	val, ok := args["mode"]

	if !ok {
		return "", fmt.Errorf("Display history 'mode' not found")
	}

	mode, ok := val.(string)

	if !ok {
		return "", fmt.Errorf("Display history 'mode' argument should be a string")
	}

	if mode == "linear" {
		return cdh.currentDag.String(), nil
	} else if mode == "dag" {

		depth, ok := args["depth"]

		if !ok {
			return "", fmt.Errorf("'depth' argument should be specified in dag mode")
		}

		dagDepth, ok := depth.(int)

		if !ok {
			return "", fmt.Errorf("'depth' argument should be an integer")
		}

		return cdh.currentDag.ToASCIITree(dagDepth), nil
	}

	return "", fmt.Errorf("Display history mode should be 'linear' or 'dag'")

}
