package keysitematch

import (
	"github.com/kevin-zx/site-info-crawler/sitethrougher"
	"math"
	"strings"
)

type KeywordMatchURL struct {
	MatchUrl          string
	TitleAllMatch     bool
	TitleMatchRate    float64
	H1AllMatch        bool
	H1MatchRate       float64
	ContentAllMatch   bool
	ContentMatchRate  float64
	HrefTextAllMatch  bool
	HrefTextMatchRate float64
	QuoteCount        int
	HomePageMatchType string
	Link              *sitethrougher.SiteLinkInfo
}

type Result struct {
	TitleMatchCount     int
	H1MatchCount        int
	ContentMatchCount   int
	MaxContentMatchRate float64
	HrefTextMatchCount  int
	MatchIndex          int
	HomePageMatchRate   float64
	HomePageMatchType   string
}

func Match(si *sitethrougher.SiteInfo, keywords []string) map[string]*Result {
	sitethrougher.FillSiteLinksDetailHrefText(si)
	kms := DetailMatch(si, keywords)
	km := make(map[string]*Result)
	for k, ms := range kms {
		r := &Result{
			TitleMatchCount:    0,
			H1MatchCount:       0,
			ContentMatchCount:  0,
			HrefTextMatchCount: 0,
			MatchIndex:         0,
			HomePageMatchRate:  0,
		}
		km[k] = r
		for _, urlM := range ms {
			if r.MaxContentMatchRate < urlM.ContentMatchRate {
				r.MaxContentMatchRate = urlM.ContentMatchRate
			}
			if urlM.Link.PageType == sitethrougher.PageTypeHome {
				r.HomePageMatchRate = urlM.TitleMatchRate
				if urlM.TitleAllMatch {
					r.HomePageMatchType = "全匹配"
				} else {
					if urlM.TitleMatchRate > 0 {
						r.HomePageMatchType = "非全匹配"
					} else {
						r.HomePageMatchType = "未匹配"
					}
				}
			}

			isAdd := false
			if urlM.HrefTextAllMatch {
				r.HrefTextMatchCount += 1
				r.MatchIndex += urlM.Link.QuoteCount * 4
				isAdd = true
			} else {
				r.MatchIndex += int(float64(urlM.Link.QuoteCount) * math.Pow(urlM.HrefTextMatchRate/2.0, 1))
			}
			if urlM.TitleAllMatch {
				r.TitleMatchCount += 1
				if !isAdd {
					r.MatchIndex += urlM.Link.QuoteCount * 3
					isAdd = true
				}
			} else {
				r.MatchIndex += int(float64(urlM.Link.QuoteCount) * math.Pow(urlM.TitleMatchRate/2.0, 2))
			}
			if urlM.H1AllMatch {
				r.H1MatchCount += 1
				if !isAdd {
					r.MatchIndex += urlM.Link.QuoteCount * 2
					isAdd = true
				}
			} else {
				r.MatchIndex += int(float64(urlM.Link.QuoteCount) * math.Pow(urlM.H1MatchRate/2.0, 3))
			}
			if urlM.ContentAllMatch {
				r.ContentMatchCount += 1
				if !isAdd {
					r.MatchIndex += urlM.Link.QuoteCount
					isAdd = true
				}
			} else {
				r.MatchIndex += int(float64(urlM.Link.QuoteCount) * math.Pow(urlM.H1MatchRate/2.0, 5))
			}
		}

	}
	return km
}

func DetailMatch(si *sitethrougher.SiteInfo, keywords []string) map[string][]*KeywordMatchURL {
	km := make(map[string][]*KeywordMatchURL)
	for _, link := range si.SiteLinks {
		for _, keyword := range keywords {
			kmu := &KeywordMatchURL{
				MatchUrl:         link.AbsURL,
				TitleAllMatch:    false,
				TitleMatchRate:   0,
				H1AllMatch:       false,
				H1MatchRate:      0,
				ContentAllMatch:  false,
				ContentMatchRate: 0,
				QuoteCount:       link.QuoteCount,
			}
			kmu.Link = link
			title := ""
			if link.WebPageSeoInfo != nil {
				title = link.WebPageSeoInfo.Title
			}
			kmu.TitleMatchRate, kmu.TitleAllMatch = CalculateMatchRate(keyword, title)
			kmu.H1MatchRate, kmu.H1AllMatch = CalculateMatchRate(keyword, link.H1)
			kmu.ContentMatchRate, kmu.ContentAllMatch = CalculateMatchRate(keyword, link.InnerText)

			max := 0
			hrefText := ""
			for s,href := range link.DetailHrefTexts {
				if len(s) < len(keyword) || len(s)-len(keyword) >= 30 {
					continue
				}
				if href.Count > max {
					max = href.Count
					hrefText= s
				}

				//mr, hrMatch := CalculateMatchRate(keyword, s)
				//if kmu.HrefTextMatchRate <= mr {
				//	kmu.HrefTextMatchRate = mr
				//}
				//kmu.HrefTextAllMatch = kmu.HrefTextAllMatch || hrMatch
				//if kmu.HrefTextAllMatch {
				//	break
				//}
			}

			kmu.HrefTextMatchRate, kmu.HrefTextAllMatch = CalculateMatchRate(keyword, hrefText)

			if _, ok := km[keyword]; ok {
				km[keyword] = append(km[keyword], kmu)
			} else {
				km[keyword] = []*KeywordMatchURL{kmu}
			}
		}
	}
	return km
}

func CalculateMatchRate(key string, matchTxt string) (float64, bool) {
	key = strings.ToLower(key)
	matchTxt = strings.ToLower(matchTxt)
	allMatch := strings.Contains(matchTxt, key)
	if allMatch {
		return 1, true
	}
	kp := strings.Split(key, "")
	kl := float64(len(kp))
	ml := 0.0
	for _, s := range kp {
		if strings.Contains(matchTxt, s) {
			ml += 1.0
		}
	}
	return ml / kl, false
}
