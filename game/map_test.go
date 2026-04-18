package main

import (
	"reflect"
	"testing"
)

func TestMapStuff(t *testing.T) {

	theMap := defaultMap()

	theMap.writeMap(MAP_PATH + "default.json")

	alsoTheMap := loadMap(MAP_PATH + "default.json")

	// u can't easily compare closures so just dont
	theMap.dangerGradient = nil
	alsoTheMap.dangerGradient = nil

	if !reflect.DeepEqual(theMap, alsoTheMap) {
		t.Errorf("The loaded map does not match the written one\n%v\n%v\n", theMap, alsoTheMap)
	}
}
