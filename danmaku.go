package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// 弹幕画布参数
const (
	stageWidth     = 1920
	stageHeight    = 1080
	fontName       = "Microsoft YaHei"
	rollDuration   = 8.0  // 滚动弹幕显示时长（秒）
	fixedDuration  = 5.0  // 顶/底部固定弹幕显示时长（秒）
	defaultLineH   = 32.0 // 默认行高
	textAlpha      = "30" // 文字透明度（00=不透明，FF=全透明）
)

type danmakuComment struct {
	time     float64 // 出现时间（秒）
	mode     int     // 1/2/3=滚动 4=底部 5=顶部 6=逆向滚动
	fontSize int
	color    int // RGB int
	text     string
}

type bilibiliXML struct {
	XMLName xml.Name `xml:"i"`
	Items   []struct {
		P    string `xml:"p,attr"`
		Text string `xml:",chardata"`
	} `xml:"d"`
}

func parseDanmakuXML(data []byte) ([]danmakuComment, error) {
	var doc bilibiliXML
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	var comments []danmakuComment
	for _, item := range doc.Items {
		parts := strings.Split(item.P, ",")
		if len(parts) < 4 {
			continue
		}
		t, _ := strconv.ParseFloat(parts[0], 64)
		m, _ := strconv.Atoi(parts[1])
		fs, _ := strconv.Atoi(parts[2])
		c, _ := strconv.Atoi(parts[3])
		if fs == 0 {
			fs = 25
		}
		text := strings.TrimSpace(item.Text)
		text = strings.ReplaceAll(text, "\n", " ")
		text = strings.ReplaceAll(text, "{", "(")
		text = strings.ReplaceAll(text, "}", ")")
		if text == "" {
			continue
		}
		comments = append(comments, danmakuComment{
			time: t, mode: m, fontSize: fs, color: c, text: text,
		})
	}
	sort.Slice(comments, func(i, j int) bool {
		return comments[i].time < comments[j].time
	})
	return comments, nil
}

func formatASSTime(sec float64) string {
	if sec < 0 {
		sec = 0
	}
	h := int(sec) / 3600
	m := (int(sec) % 3600) / 60
	s := int(sec) % 60
	cs := int((sec - float64(int(sec))) * 100)
	return fmt.Sprintf("%d:%02d:%02d.%02d", h, m, s, cs)
}

// B站 RGB → ASS BGR（ASS 用 BGR 顺序，前面再加透明度字节）
func rgbToASSColor(rgb int) string {
	b := rgb & 0xff
	g := (rgb >> 8) & 0xff
	r := (rgb >> 16) & 0xff
	return fmt.Sprintf("&H%s%02X%02X%02X", textAlpha, b, g, r)
}

// 估算文字宽度（粗略：一个字符约等于 fontSize × 0.6）
func estimateTextWidth(text string, fontSize int) int {
	width := 0
	for _, r := range text {
		if r > 127 {
			width += fontSize // 中文/全角字符占 fontSize 宽
		} else {
			width += int(float64(fontSize) * 0.6) // ASCII 半宽
		}
	}
	return width
}

// 找一个空闲车道。返回车道索引（0 = 最上面）
// 没有空闲就选最早释放的（允许重叠）
func findLane(freeTimes []float64, t float64) int {
	for i, ft := range freeTimes {
		if t >= ft {
			return i
		}
	}
	earliest := 0
	for i := range freeTimes {
		if freeTimes[i] < freeTimes[earliest] {
			earliest = i
		}
	}
	return earliest
}

// ConvertDanmakuXMLToASS 把 B站/A站 XML 弹幕转成 ASS 字幕字符串
func ConvertDanmakuXMLToASS(xmlData []byte) (string, error) {
	comments, err := parseDanmakuXML(xmlData)
	if err != nil {
		return "", err
	}
	if len(comments) == 0 {
		return "", fmt.Errorf("没有有效弹幕")
	}

	lineH := defaultLineH
	maxRollLanes := int(float64(stageHeight) / lineH)
	maxFixedLanes := maxRollLanes / 3 // 顶部/底部各占约 1/3 屏

	rollLanes := make([]float64, maxRollLanes)
	topLanes := make([]float64, maxFixedLanes)
	botLanes := make([]float64, maxFixedLanes)

	var dialogues []string

	for _, c := range comments {
		textWidth := estimateTextWidth(c.text, c.fontSize)
		colorTag := ""
		if c.color != 0xFFFFFF && c.color != 0 {
			colorTag = "\\c" + strings.Replace(rgbToASSColor(c.color), textAlpha, "00", 1)
		}

		var dialogue string
		switch c.mode {
		case 1, 2, 3: // 滚动
			laneIdx := findLane(rollLanes, c.time)
			y := int(float64(laneIdx)*lineH + lineH/2)
			startX := stageWidth + textWidth/2
			endX := -textWidth / 2
			// 该车道在弹幕走完之前不能再用
			rollLanes[laneIdx] = c.time + rollDuration*float64(textWidth)/float64(stageWidth+textWidth)
			startStr := formatASSTime(c.time)
			endStr := formatASSTime(c.time + rollDuration)
			dialogue = fmt.Sprintf("Dialogue: 0,%s,%s,Roll,,0,0,0,,{\\move(%d,%d,%d,%d)%s}%s",
				startStr, endStr, startX, y, endX, y, colorTag, c.text)

		case 5: // 顶部
			laneIdx := findLane(topLanes, c.time)
			y := int(float64(laneIdx)*lineH + lineH/2)
			topLanes[laneIdx] = c.time + fixedDuration
			startStr := formatASSTime(c.time)
			endStr := formatASSTime(c.time + fixedDuration)
			dialogue = fmt.Sprintf("Dialogue: 0,%s,%s,Top,,0,0,0,,{\\an8\\pos(%d,%d)%s}%s",
				startStr, endStr, stageWidth/2, y, colorTag, c.text)

		case 4: // 底部
			laneIdx := findLane(botLanes, c.time)
			y := stageHeight - int(float64(laneIdx)*lineH+lineH/2)
			botLanes[laneIdx] = c.time + fixedDuration
			startStr := formatASSTime(c.time)
			endStr := formatASSTime(c.time + fixedDuration)
			dialogue = fmt.Sprintf("Dialogue: 0,%s,%s,Bottom,,0,0,0,,{\\an2\\pos(%d,%d)%s}%s",
				startStr, endStr, stageWidth/2, y, colorTag, c.text)

		default:
			continue
		}
		dialogues = append(dialogues, dialogue)
	}

	var sb strings.Builder
	sb.WriteString("[Script Info]\n")
	sb.WriteString("ScriptType: v4.00+\n")
	sb.WriteString(fmt.Sprintf("PlayResX: %d\n", stageWidth))
	sb.WriteString(fmt.Sprintf("PlayResY: %d\n", stageHeight))
	sb.WriteString("Aspect Ratio: 16:9\n")
	sb.WriteString("Collisions: Normal\n")
	sb.WriteString("WrapStyle: 2\n")
	sb.WriteString("ScaledBorderAndShadow: yes\n\n")

	sb.WriteString("[V4+ Styles]\n")
	sb.WriteString("Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding\n")
	styleFmt := "Style: %s,%s,25,&H00FFFFFF,&H00FFFFFF,&H80000000,&H80000000,0,0,0,0,100,100,0,0,1,1,0,%d,0,0,0,1\n"
	sb.WriteString(fmt.Sprintf(styleFmt, "Roll", fontName, 7))
	sb.WriteString(fmt.Sprintf(styleFmt, "Top", fontName, 8))
	sb.WriteString(fmt.Sprintf(styleFmt, "Bottom", fontName, 2))
	sb.WriteString("\n")

	sb.WriteString("[Events]\n")
	sb.WriteString("Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n")
	for _, d := range dialogues {
		sb.WriteString(d)
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// ConvertDanmakuFilesInDir 扫描目录里所有 .xml 弹幕，转成同名 .ass，原 xml 删除
func ConvertDanmakuFilesInDir(dir string) (converted []string, errs []error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.xml"))
	if err != nil {
		return nil, []error{err}
	}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %v", filepath.Base(f), err))
			continue
		}
		ass, err := ConvertDanmakuXMLToASS(data)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %v", filepath.Base(f), err))
			continue
		}
		assPath := strings.TrimSuffix(f, filepath.Ext(f)) + ".ass"
		if err := os.WriteFile(assPath, []byte(ass), 0644); err != nil {
			errs = append(errs, fmt.Errorf("%s: %v", filepath.Base(f), err))
			continue
		}
		os.Remove(f)
		converted = append(converted, filepath.Base(assPath))
	}
	return converted, errs
}

func isBiliOrAcFun(url string) bool {
	low := strings.ToLower(url)
	for _, d := range []string{"bilibili.com", "b23.tv", "acfun.cn"} {
		if strings.Contains(low, d) {
			return true
		}
	}
	return false
}
