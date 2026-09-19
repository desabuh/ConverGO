package repository

import (
	"os"
	"testing"

	"github.com/desabuh/convergo/cvrdt"
	"github.com/desabuh/convergo/utils"
	"github.com/stretchr/testify/assert"
)

var SITE_ID = "1"
var DOMAIN_PATH = "./REPO_TEST_DOMAIN/"

// assume the CvrdtRepository is based on Woot
var CRDTfactory = func() utils.ObservableState[cvrdt.CvRDTState, string] {
	return cvrdt.NewWootCvrdtWithView(SITE_ID)
}
var CRDTadapter = CvrdtToRepoFilesAdapter[cvrdt.WootOperation]{}

var TEST_FILE = "file.txt"

func GetOp(opType string, content string, pos int) TextualFileOperation {
	return TextualFileOperation{
		SourceId: SITE_ID,
		FilePath: TEST_FILE,
		OpType:   opType,
		Pos:      pos,
		Content:  content,
	}
}

func TestCreationSimpleOpbasedRepo(t *testing.T) {
	var SITE_ID = "1"
	var POS = 0
	var CONTENT = "hi"
	var OPTYPE = "insert"

	defer os.RemoveAll(DOMAIN_PATH)

	CRDTfactory := func() utils.ObservableState[cvrdt.CvRDTState, string] {
		return cvrdt.NewWootCvrdtWithView(SITE_ID)
	}

	CRDTadapter := CvrdtToRepoFilesAdapter[cvrdt.WootOperation]{}

	repo := GetNewCvrdtRepository(SITE_ID, DOMAIN_PATH, CRDTfactory, CRDTadapter)
	defer repo.ShutDownRepo()

	op := GetOp(OPTYPE, CONTENT, POS)

	status, err := repo.InsertCRDTOpInRepo(op)

	assert.Nil(t, err)
	//operation should have id SITE_ID + "_1"
	assert.Equal(t, SITE_ID+"_1", status.Id)
	assert.Equal(t, CONTENT, status.TargetChar)

	//this operation created a new file
	assert.True(t, status.NewFile)

	//ensure also fileregistry has the updated state content
	content, err := repo.GetFileContent(TEST_FILE)

	assert.Nil(t, err)

	assert.Equal(t, CONTENT, content)

	res, err := repo.DisplayFileHistory(TEST_FILE, "linear", map[string]any{})

	assert.Nil(t, err)

	//the resulting DAG should have only 1 node and no edges

	expectedRes := "DAG with 1 nodes and 0 edges\n\n" +
		"Nodes:\n" +
		"  1_1: 0 parents, 0 children\n\n" +
		"Edges:\n"

	assert.Equal(t, res, expectedRes)

}

func TestCreationMultipleOperations(t *testing.T) {
	defer os.RemoveAll(DOMAIN_PATH)

	repo := GetNewCvrdtRepository(SITE_ID, DOMAIN_PATH, CRDTfactory, CRDTadapter)
	defer repo.ShutDownRepo()

	op1 := GetOp("insert", "hi", 0)
	op2 := GetOp("delete", "h", 0)
	op3 := GetOp("insert", "bye", 0)

	operations := []TextualFileOperation{
		op1,
		op2,
		op3,
	}

	for _, op := range operations {
		_, err := repo.InsertCRDTOpInRepo(op)
		assert.Nil(t, err)
	}

	content, err := repo.GetFileContent(TEST_FILE)

	assert.Nil(t, err)

	assert.Equal(t, "bye", content)

	res, err := repo.DisplayFileHistory(TEST_FILE, "linear", map[string]any{})

	assert.Nil(t, err)

	//the three operation should have 3 nodes and be sequential (two happened before edges)
	expectedRes := "DAG with 3 nodes and 2 edges\n\n" +
		"Nodes:\n" +
		"  1_1: 0 parents, 1 children\n" +
		"  1_2: 1 parents, 1 children\n" +
		"  1_3: 1 parents, 0 children\n\n" +
		"Edges:\n" +
		"  1_1 → 1_2\n" +
		"  1_2 → 1_3\n"

	assert.Equal(t, expectedRes, res)

	res, err = repo.DisplayFileHistory(TEST_FILE, "dag", map[string]any{"depth": 10})

	assert.Nil(t, err)

	expectedRes =
		"\n┌─ DAG: 3 operations, 2 causal edges\n" +
			"│\n" +
			"├─ 1_1 [Insertion] 'hi' → 1_2\n" +
			"│\n" +
			"│  ├─ 1_2 [Deletion] 'hi' → 1_3\n" +
			"│  │\n" +
			"│  │  └─ 1_3 [Insertion] 'bye'\n"

	assert.Equal(t, expectedRes, res)
}
