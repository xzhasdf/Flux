package main

import (
	"context"
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "Flux",
		Width:  1200,
		Height: 820,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		// 有任务在跑时关窗需二次确认；退出后任务在历史记录中标记为「已中断」，可继续
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			if !app.HasActiveTasks() {
				return false
			}
			sel, err := runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
				Type:          runtime.QuestionDialog,
				Title:         "任务正在进行",
				Message:       "还有下载/转码任务未完成，现在退出会中断任务。\n已下载的部分会保留，下次可在「历史记录」中继续。\n\n确定要退出吗？",
				Buttons:       []string{"仍要退出", "取消"},
				DefaultButton: "取消",
				CancelButton:  "取消",
			})
			if err != nil {
				return false
			}
			// Windows 的 Question 对话框忽略自定义按钮，返回 "Yes"/"No"
			return sel != "仍要退出" && sel != "Yes"
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
