package main

import "testing"

func TestExtractTitlePresentLowerCase(t *testing.T) {
	body := "this is a dummy<title>test page</title> to make sure title extracting works great"

	title := extractTitle(body)
	if title != "test page" {
		t.Errorf("Found title: %v is not 'test page'", title)
	}
}

func TestExtractTitlePresentUpperCaseAndPreserveCase(t *testing.T) {
	body := "this is A dummy<TITLE>test PaGe</TITLE> to MAKE sure title EXTRACTING works great"

	title := extractTitle(body)
	if title != "test PaGe" {
		t.Errorf("Found title: %v is not 'test PaGe'", title)
	}
}

func TestExtractTitleNotPresent(t *testing.T) {
	body := "this is a dummy to make sure title extracting works great"

	title := extractTitle(body)
	if title != "" {
		t.Errorf("Found title but no one are present")
	}
}