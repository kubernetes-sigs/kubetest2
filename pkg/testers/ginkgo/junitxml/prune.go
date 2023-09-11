/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package junitxml

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"sigs.k8s.io/kubetest2/pkg/artifacts"
)

const maxTextSize = 1 // one MB

func Clean() {
	xmlFile := filepath.Join(artifacts.BaseDir(), "junit_01.xml")
	_, err := os.Stat(xmlFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Printf("junit xml file %s not present.\n", xmlFile)
		} else {
			fmt.Printf("error opening junit xml file %s : %s\n", xmlFile, err)
		}
		return
	}
	fmt.Printf("processing junit xml file : %s\n", xmlFile)
	xmlReader, err := os.Open(xmlFile)
	if err != nil {
		return
	}
	defer xmlReader.Close()
	suites, err := fetchXML(xmlReader) // convert MB into bytes (roughly!)
	if err != nil {
		fmt.Printf("error fetching xml : %s\n", err)
		return
	}

	pruneXML(suites, maxTextSize*1e6) // convert MB into bytes (roughly!)

	xmlWriter, err := os.OpenFile(xmlFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		fmt.Printf("error opening file to write xml : %s\n", err)
		return
	}
	defer xmlWriter.Close()
	err = streamXML(xmlWriter, suites)
	if err != nil {
		fmt.Printf("error streaming xml to file: %s\n", err)
		return
	}
	fmt.Println("done.")
}

func pruneXML(suites *JUnitTestSuites, maxBytes int) {
	for _, suite := range suites.Suites {
		for _, testcase := range suite.TestCases {
			if testcase.SkipMessage != nil {
				if len(testcase.SkipMessage.Message) > maxBytes {
					fmt.Printf("clipping skip message in test case : %s\n", testcase.Name)
					head := testcase.SkipMessage.Message[:maxBytes/2]
					tail := testcase.SkipMessage.Message[len(testcase.SkipMessage.Message)-maxBytes/2:]
					testcase.SkipMessage.Message = head + "[...clipped...]" + tail
				}
			}
			if testcase.Failure != nil {
				if len(testcase.Failure.Contents) > maxBytes {
					fmt.Printf("clipping failure message in test case : %s\n", testcase.Name)
					head := testcase.Failure.Contents[:maxBytes/2]
					tail := testcase.Failure.Contents[len(testcase.Failure.Contents)-maxBytes/2:]
					testcase.Failure.Contents = head + "[...clipped...]" + tail
				}
			}
		}
	}
}

func fetchXML(xmlReader io.Reader) (*JUnitTestSuites, error) {
	decoder := xml.NewDecoder(xmlReader)
	var suites JUnitTestSuites
	err := decoder.Decode(&suites)
	if err != nil {
		return nil, err
	}
	return &suites, nil
}

func streamXML(writer io.Writer, in *JUnitTestSuites) error {
	_, err := writer.Write([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"))
	if err != nil {
		return err
	}
	encoder := xml.NewEncoder(writer)
	encoder.Indent("", "\t")
	err = encoder.Encode(in)
	if err != nil {
		return err
	}
	return encoder.Flush()
}
