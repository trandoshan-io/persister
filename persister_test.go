package main

import (
	"fmt"
	"testing"
	"time"
)

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

func TestComputePathNoExtension(t *testing.T) {
	currentTime := time.Now()
	path := computePath("https://private.creekorful.fr/something/strange/", currentTime)

	if path != fmt.Sprintf("private.creekorful.fr/something/strange/%d", currentTime.Unix()) {
		t.Errorf("Computed path is wrong")
	}
}

func TestComputePathWithExtension(t *testing.T) {
	currentTime := time.Now()
	path := computePath("http://private.creekorful.fr/something/strange/index.php", currentTime)

	if path != fmt.Sprintf("private.creekorful.fr/something/strange/index.php/%d", currentTime.Unix()) {
		t.Errorf("Computed path is wrong")
	}
}
