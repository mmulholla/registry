package tests

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"reflect"

	devfilepkg "github.com/devfile/parser/pkg/devfile"
	"github.com/devfile/parser/pkg/devfile/parser"
	v2 "github.com/devfile/parser/pkg/devfile/parser/data/v2"
)

// Users struct which contains
// an array of users
type TestJson struct {
	SchemaFile    string   `json:"SchemaFile"`
	SchemaVersion string   `json:"SchemaVersion"`
	Tests         []string `json:"Tests"`
}

// User struct which contains a name
// a type and a list of social links
type TestsToRun struct {
	Tests []TestToRun `json:"Tests"`
}

// User struct which contains a name
// a type and a list of social links
type TestToRun struct {
	FileName      string   `json:"FileName"`
	Disabled       bool     `json:"Disabled"`
	ExpectOutcome string   `json:"ExpectOutcome"`
	Files         []string `json:"Files"`
}

const testDir = "../../"
const jsonDir = "./json/v200/"
const tempRootDir = "./tmp/"
const tempDir = "./tmp/v200/"

const logErrorOnly = false

func Test_API_200(t *testing.T) {

	// Clear the temp directory if it exists
	if _, err := os.Stat(tempRootDir); !os.IsNotExist(err) {
		os.RemoveAll(tempRootDir)
	}
	os.Mkdir(tempRootDir, 0755)
	os.Mkdir(tempDir, 0755)

	// Read the content of the json directory to find test files
	files, err := ioutil.ReadDir(jsonDir)
	if err != nil {
		t.Fatalf("Error finding test json files in : %s :  %v", jsonDir, err)
	}
	combinedTests := 0
	combinedPasses := 0
	for _, testJsonFile := range files {

		// t.Logf("Found file: %s",testJsonFile.Name());
		// if the file ends with -test.json it can be processed
		if strings.HasSuffix(testJsonFile.Name(),"-tests.json") {

			// Open the json file which defines the tests to run
			testJson, err := os.Open(filepath.Join(jsonDir, testJsonFile.Name()))
			if err != nil {
				t.Errorf("  FAIL : Failed to open %s : %s", testJsonFile.Name(), err)
				continue
			}

			// Read contents of the json file which defines the tests to run
			byteValue, err := ioutil.ReadAll(testJson)
			if err != nil {
				t.Errorf("FAIL : failed to read : %s : %v", testJsonFile.Name(), err)
			}

			var testsToRunContent TestsToRun

			// Unmarshall the contents of the json file which defines the tests to run for each test
			err = json.Unmarshal(byteValue, &testsToRunContent)
			if err != nil {
				t.Fatalf("FAIL : failed to unmarshal : %s : %v", testJsonFile.Name(), err)
				continue
			}

			testJson.Close()

			passTests := 0
			totalTests := 0

			// For each test defined in the test file
			for i := 0; i < len(testsToRunContent.Tests); i++ {

				if !testsToRunContent.Tests[i].Disabled {
								
					totalTests++

					// Open the file to containe the generated test yaml
					f, err := os.OpenFile(filepath.Join(tempDir, testsToRunContent.Tests[i].FileName), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
					if err != nil {
						t.Errorf("FAIL : Failed to open %s : %v", filepath.Join(tempDir, testsToRunContent.Tests[i].FileName), err)
						continue
					}

					f.WriteString("schemaVersion: \"2.0.0\"\n")

					testYamlComplete := true
					// Now add each of the yaml sippets used the make the yaml file for test
					for j := 0; j < len(testsToRunContent.Tests[i].Files); j++ {
						// Read the snippet
						data, err := ioutil.ReadFile(filepath.Join(testDir, testsToRunContent.Tests[i].Files[j]))
						if err != nil {
							t.Errorf("FAIL: failed reading %s: %v", filepath.Join(testDir, testsToRunContent.Tests[i].Files[j]), err)
							testYamlComplete = false
							continue
						}
						if j > 0 {
							// Ensure approproate line breaks
							f.WriteString("\n")
						}

						// Add snippet to yaml file
						f.Write(data)
					}

					if !testYamlComplete {
						f.Close()
						continue
					}

					devfileName := filepath.Join(tempDir, testsToRunContent.Tests[i].FileName);
					// Read the created yaml file, ready for converison to json
					//data, err := ioutil.ReadFile(filepath.Join(testTempDir, testsToRunContent.Tests[i].FileName))
					devfile, err := ParseDevfile(devfileName)
					if err != nil {
						if testsToRunContent.Tests[i].ExpectOutcome == "PASS" {
							t.Errorf("  FAIL : %s : Validate failure : %s", testsToRunContent.Tests[i].FileName, err)
						} else if testsToRunContent.Tests[i].ExpectOutcome == "" { 
							t.Errorf("  FAIL : %s : No expected ouctome was set : %s  got : %s", testsToRunContent.Tests[i].FileName, testsToRunContent.Tests[i].ExpectOutcome, err.Error())						
						} else if !strings.Contains(err.Error(), testsToRunContent.Tests[i].ExpectOutcome) {
							t.Errorf("  FAIL : %s : Did not fail as expected : %s  got : %s", testsToRunContent.Tests[i].FileName, testsToRunContent.Tests[i].ExpectOutcome, err.Error())
						} else {
							passTests++
							if !logErrorOnly {
								t.Logf("PASS : %s : %s", testsToRunContent.Tests[i].FileName, testsToRunContent.Tests[i].ExpectOutcome)
							}	
						}
					} else if testsToRunContent.Tests[i].ExpectOutcome == "" { 
						t.Errorf("  FAIL : %s : devfile was valid - No expected ouctome was set.", testsToRunContent.Tests[i].FileName)							
					} else if testsToRunContent.Tests[i].ExpectOutcome != "PASS" {
						t.Errorf("  FAIL : %s : devfile was valid - Expected Error not found :  %s", testsToRunContent.Tests[i].FileName, testsToRunContent.Tests[i].ExpectOutcome)
					} else {
						if !logErrorOnly {
							devdata := devfile.Data
							if (reflect.TypeOf(devdata) == reflect.TypeOf(&v2.DevfileV2{})) {
								d := devdata.(*v2.DevfileV2)
								t.Logf("PASS : %s : Schema Version found %s",devfileName,d.SchemaVersion)
							}
						}
						passTests++
						//for _, component := range devfile.Data.GetComponents() {
						//	if component.Container != nil {
						//		fmt.Println(component.Container.Image)
						//}
						//}
				
						//for _, command := range devfile.Data.GetCommands() {
						//	if command.Exec != nil {
						//		fmt.Println(command.Exec.Group.Kind)
						//	}
						//}
					}
				}
			}
			combinedTests += totalTests
			combinedPasses += passTests 

		}
	}

	if combinedTests != combinedPasses {
		t.Errorf("OVERALL FAIL : %d of %d tests failed.", (combinedTests - combinedPasses), combinedTests)
	} else {
		t.Logf("OVERALL PASS : %d of %d tests passed.", combinedPasses, combinedTests)
	}
}

//ParseDevfile to parse devfile from library
func ParseDevfile(devfileLocation string) (parser.DevfileObj, error) {

	devfile, err := devfilepkg.ParseAndValidate(devfileLocation)
	return devfile, err
}
