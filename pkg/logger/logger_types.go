package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Level zapcore.Level

type Logger zap.Logger

type Field = zap.Field

var String = zap.String
var Strings = zap.Strings
var Error = zap.Error
var ByteString = zap.ByteString
var Int = zap.Int
var Int32 = zap.Int32
var Int64 = zap.Int64
var Any = zap.Any
var Time = zap.Time

var NewNop = zap.NewNop