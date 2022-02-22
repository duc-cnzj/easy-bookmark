package main

import (
	"github.com/fatih/color"
	"strings"
)

func Debug(v ...interface{})  {
	color.White(strings.Repeat("%v ", len(v)), v...)
}

func Debugf(fmt string, v ...interface{})  {
	color.White(fmt, v...)
}

func Info(v ...interface{})  {
	color.Green(strings.Repeat("%v ", len(v)), v...)
}

func Infof(fmt string, v ...interface{})  {
	color.Green(fmt, v...)
}

func Warn(v ...interface{})  {
	color.Yellow(strings.Repeat("%v ", len(v)), v...)
}

func Warnf(fmt string, v ...interface{})  {
	color.Yellow(fmt, v...)
}

func Error(v ...interface{})  {
	color.Red(strings.Repeat("%v ", len(v)), v...)
}

func Errorf(fmt string, v ...interface{})  {
	color.Red(fmt, v...)
}
