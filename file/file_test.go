package file

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/desabuh/convergo/cvrdt"
	"github.com/stretchr/testify/assert"
)

func TestNewFileContextOperation(t *testing.T) {
	const SITE_ID = "1"
	const START_POS = 0
	const CONTENT = "Hi"

	tmpFile, err := os.CreateTemp("./", "tmpfile-*.txt")
	if err != nil {
		log.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	fileCtx := GetNewFileContext("TEST_DOMAIN", "path", cvrdt.NewWootCvrdtWithView(SITE_ID), tmpFile, 100*time.Millisecond)

	fileCtx.TrackState()
	defer fileCtx.UntrackState()

	fileCtx.UpdateState(
		cvrdt.GetNewStateFromOp(
			cvrdt.CreateNewLocalOp(
				cvrdt.Insertion,
				START_POS,
				CONTENT,
				SITE_ID,
			),
		),
	)

	var state cvrdt.CvRDTState = fileCtx.GetStateCopy().readOnlyState

	time.Sleep(300 * time.Millisecond)

	operations := make([]cvrdt.CRDTOperation, 0, len(state))
	for _, c := range state {
		operations = append(operations, c)
	}

	assert.Equal(t, 1, len(operations))

	assert.Equal(t, operations[0].Char(), CONTENT)

	tmpFile.Seek(0, 0)
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		log.Fatalf("Failed to read temp file: %v", err)
	}

	assert.Equal(t, CONTENT, string(content))
}
