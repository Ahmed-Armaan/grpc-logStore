package main

import (
	pb "github.com/Ahmed-Armaan/grpc-logStore.git/logStore/logStore"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Log struct {
	Message   string                `json:"message"`
	Timestamp timestamppb.Timestamp `json:"timestamp"`
	Details   string                `json:"details"`
	Level     pb.Loglevel           `json:"level"`
}

type LogNode struct {
	currLog *Log
	prev    *LogNode
	next    *LogNode
}

func ListInit(log *Log) *LogNode {
	var currLog *Log
	if log == nil {
		currLog = &Log{
			Message:   "List begin",
			Timestamp: *timestamppb.Now(),
			Details:   "",
			Level:     -1,
		}
	} else {
		currLog = log
	}

	newList := &LogNode{
		currLog: currLog,
		prev:    nil,
		next:    nil,
	}
	return newList
}

func (list *LogNode) InsertLog(log *Log) *LogNode {
	newNode := ListInit(log)
	list.next = newNode
	newNode.prev = list
	return newNode
}

func (list *LogNode) GetLog() (*LogNode, *Log) {
	return list.prev, list.currLog
}

func (list *LogNode) DeleteAll() *LogNode {
	curr := list
	for curr != nil && curr.currLog.Message != "List begin" {
		prev := curr.prev
		if prev != nil {
			prev.next = nil
		}
		curr.prev = nil
		curr.next = nil
		curr.currLog = nil
		curr = prev
	}

	return curr
}
