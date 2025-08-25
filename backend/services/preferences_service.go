package services

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"tinyrdm/backend/consts"
	storage2 "tinyrdm/backend/storage"
	"tinyrdm/backend/types"
	"tinyrdm/backend/utils/coll"
	convutil "tinyrdm/backend/utils/convert"
	sliceutil "tinyrdm/backend/utils/slice"

	"github.com/adrg/sysfont"
	runtime2 "github.com/wailsapp/wails/v2/pkg/runtime"
)

type preferencesService struct {
	pref          *storage2.PreferencesStorage
	clientVersion string
}

var preferences *preferencesService
var oncePreferences sync.Once

func Preferences() *preferencesService {
	if preferences == nil {
		oncePreferences.Do(func() {
			preferences = &preferencesService{
				pref:          storage2.NewPreferences(),
				clientVersion: "",
			}
		})
	}
	return preferences
}

func (p *preferencesService) GetPreferences() (resp types.JSResp) {
	resp.Data = p.pref.GetPreferences()
	resp.Success = true
	return
}

func (p *preferencesService) SetPreferences(pf types.Preferences) (resp types.JSResp) {
	err := p.pref.SetPreferences(&pf)
	if err != nil {
		resp.Msg = err.Error()
		return
	}

	p.UpdateEnv()
	resp.Success = true
	return
}

func (p *preferencesService) UpdatePreferences(value map[string]any) (resp types.JSResp) {
	err := p.pref.UpdatePreferences(value)
	if err != nil {
		resp.Msg = err.Error()
		return
	}
	resp.Success = true
	return
}

func (p *preferencesService) RestorePreferences() (resp types.JSResp) {
	defaultPref := p.pref.RestoreDefault()
	resp.Data = map[string]any{
		"pref": defaultPref,
	}
	resp.Success = true
	return
}

type FontItem struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func (p *preferencesService) GetFontList() (resp types.JSResp) {
	finder := sysfont.NewFinder(nil)
	fontSet := coll.NewSet[string]()
	var fontList []FontItem
	for _, font := range finder.List() {
		if len(font.Family) > 0 && !strings.HasPrefix(font.Family, ".") && fontSet.Add(font.Family) {
			fontList = append(fontList, FontItem{
				Name: font.Family,
				Path: font.Filename,
			})
		}
	}
	sort.Slice(fontList, func(i, j int) bool {
		return fontList[i].Name < fontList[j].Name
	})
	resp.Data = map[string]any{
		"fonts": fontList,
	}
	resp.Success = true
	return
}

func (p *preferencesService) GetBuildInDecoder() (resp types.JSResp) {
	buildinDecoder := make([]string, 0, len(convutil.BuildInDecoders))
	for name, convert := range convutil.BuildInDecoders {
		if convert.Enable() {
			buildinDecoder = append(buildinDecoder, name)
		}
	}
	resp.Data = map[string]any{
		"decoder": buildinDecoder,
	}
	resp.Success = true
	return
}

func (p *preferencesService) GetLanguage() string {
	pref := p.pref.GetPreferences()
	return pref.General.Language
}

func (p *preferencesService) SetAppVersion(ver string) {
	if !strings.HasPrefix(ver, "v") {
		p.clientVersion = "v" + ver
	} else {
		p.clientVersion = ver
	}
}

func (p *preferencesService) GetAppVersion() (resp types.JSResp) {
	resp.Success = true
	resp.Data = map[string]any{
		"version": p.clientVersion,
	}
	return
}

func (p *preferencesService) SaveWindowSize(width, height int, maximised bool) {
	if maximised {
		// do not update window size if maximised state
		p.UpdatePreferences(map[string]any{
			"behavior.windowMaximised": true,
		})
	} else if width >= consts.MIN_WINDOW_WIDTH && height >= consts.MIN_WINDOW_HEIGHT {
		p.UpdatePreferences(map[string]any{
			"behavior.windowWidth":     width,
			"behavior.windowHeight":    height,
			"behavior.windowMaximised": false,
		})
	}
}

func (p *preferencesService) GetWindowSize() (width, height int, maximised bool) {
	data := p.pref.GetPreferences()
	width, height, maximised = data.Behavior.WindowWidth, data.Behavior.WindowHeight, data.Behavior.WindowMaximised
	if width <= 0 {
		width = consts.DEFAULT_WINDOW_WIDTH
	}
	if height <= 0 {
		height = consts.DEFAULT_WINDOW_HEIGHT
	}
	return
}

func (p *preferencesService) GetWindowPosition(ctx context.Context) (x, y int) {
	data := p.pref.GetPreferences()
	x, y = data.Behavior.WindowPosX, data.Behavior.WindowPosY
	width, height := data.Behavior.WindowWidth, data.Behavior.WindowHeight
	var screenWidth, screenHeight int
	if screens, err := runtime2.ScreenGetAll(ctx); err == nil {
		for _, screen := range screens {
			if screen.IsCurrent {
				screenWidth, screenHeight = screen.Size.Width, screen.Size.Height
				break
			}
		}
	}
	if screenWidth <= 0 || screenHeight <= 0 {
		screenWidth, screenHeight = consts.DEFAULT_WINDOW_WIDTH, consts.DEFAULT_WINDOW_HEIGHT
	}
	if x <= 0 || x+width > screenWidth || y <= 0 || y+height > screenHeight {
		// out of screen, reset to center
		x, y = (screenWidth-width)/2, (screenHeight-height)/2
	}
	return
}

func (p *preferencesService) SaveWindowPosition(x, y int) {
	if x > 0 || y > 0 {
		p.UpdatePreferences(map[string]any{
			"behavior.windowPosX": x,
			"behavior.windowPosY": y,
		})
	}
}

func (p *preferencesService) GetScanSize() int {
	data := p.pref.GetPreferences()
	size := data.General.ScanSize
	if size <= 0 {
		size = consts.DEFAULT_SCAN_SIZE
	}
	return size
}

func (p *preferencesService) GetDecoder() []convutil.CmdConvert {
	data := p.pref.GetPreferences()
	return sliceutil.FilterMap(data.Decoder, func(i int) (convutil.CmdConvert, bool) {
		//if !data.Decoder[i].Enable {
		//	return convutil.CmdConvert{}, false
		//}
		return convutil.CmdConvert{
			Name:       data.Decoder[i].Name,
			Auto:       data.Decoder[i].Auto,
			DecodePath: data.Decoder[i].DecodePath,
			DecodeArgs: data.Decoder[i].DecodeArgs,
			EncodePath: data.Decoder[i].EncodePath,
			EncodeArgs: data.Decoder[i].EncodeArgs,
		}, true
	})
}

type sponsorItem struct {
	Name   string   `json:"name"`
	Link   string   `json:"link"`
	Region []string `json:"region"`
}

type upgradeInfo struct {
	Version      string            `json:"version"`
	Changelog    map[string]string `json:"changelog"`
	Description  map[string]string `json:"description"`
	DownloadURl  map[string]string `json:"download_url"`
	DownloadPage map[string]string `json:"download_page"`
	Sponsor      []sponsorItem     `json:"sponsor,omitempty"`
}

func (p *preferencesService) CheckForUpdate() (resp types.JSResp) {
	resp.Success = true
	resp.Data = map[string]any{
		"version":       "v0.0.0",
		"latest":        "v0.0.0",
		"description":   "",
		"download_page": "",
		"sponsor":       "",
	}
	return
	// request latest version
	//res, err := http.Get("https://api.github.com/repos/tiny-craft/tiny-rdm/releases/latest")
	res, err := http.Get("https://redis.tinycraft.cc/client_version.json")
	if err != nil || res.StatusCode != http.StatusOK {
		resp.Msg = "network error"
		return
	}

	var respObj upgradeInfo
	err = json.NewDecoder(res.Body).Decode(&respObj)
	if err != nil {
		resp.Msg = "invalid content"
		return
	}

	// compare with current version
	resp.Success = true
	resp.Data = map[string]any{
		"version":       p.clientVersion,
		"latest":        respObj.Version,
		"description":   respObj.Description,
		"download_page": respObj.DownloadPage,
		"sponsor":       respObj.Sponsor,
	}
	return
}

// UpdateEnv Update System Environment
func (p *preferencesService) UpdateEnv() {
	if p.GetLanguage() == "zh" {
		os.Setenv("LANG", "zh_CN.UTF-8")
	} else {
		os.Unsetenv("LANG")
	}
}
