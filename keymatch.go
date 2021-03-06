package keysitematch

import (
	"github.com/kevin-zx/site-info-crawler/sitethrougher"
	"log"
	"math"
	"runtime"
	"strings"
	"sync"
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
	MatchIndex          float64
	HomePageMatchRate   float64
	HomePageMatchType   string
}

func Match(si *sitethrougher.SiteInfo, keywords []string) map[string]*Result {
	//sitethrougher.FillSiteLinksDetailHrefText(si)
	km := make(map[string]*Result)
	if len(keywords) == 0 {
		return km
	}
	var kms map[string][]*KeywordMatchURL
	max := 5000000
	l := max / len(keywords)
	if len(si.SiteLinks) < l {
		kms = DetailMatch(si.SiteLinks, keywords)
	} else {
		kms = DetailMatch(si.SiteLinks[0:l], keywords)
	}
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
				r.MatchIndex += float64(urlM.Link.QuoteCount * 4)
				isAdd = true
			} else {
				r.MatchIndex += float64(urlM.Link.QuoteCount) * math.Pow(urlM.HrefTextMatchRate/2.0, 1)
			}
			if urlM.TitleAllMatch {
				r.TitleMatchCount += 1
				if !isAdd {
					r.MatchIndex += float64(urlM.Link.QuoteCount * 3)
					isAdd = true
				}
			} else {
				r.MatchIndex += float64(urlM.Link.QuoteCount) * math.Pow(urlM.TitleMatchRate/2.0, 2)
			}
			if urlM.H1AllMatch {
				r.H1MatchCount += 1
				if !isAdd {
					r.MatchIndex += float64(urlM.Link.QuoteCount * 2)
					isAdd = true
				}
			} else {
				r.MatchIndex += float64(urlM.Link.QuoteCount) * math.Pow(urlM.H1MatchRate/2.0, 3)
			}
			if urlM.ContentAllMatch {
				r.ContentMatchCount += 1
				if !isAdd {
					r.MatchIndex += float64(urlM.Link.QuoteCount)
					isAdd = true
				}
			} else {
				r.MatchIndex += float64(urlM.Link.QuoteCount) * math.Pow(urlM.ContentMatchRate/2.0, 5)
			}
		}

	}
	return km
}

func DetailMatch(siteLinks []*sitethrougher.SiteLinkInfo, keywords []string) map[string][]*KeywordMatchURL {
	km := make(map[string][]*KeywordMatchURL)
	var tasks = make(chan parallelTask, len(keywords)+1)
	var results = make(chan result)
	core := runtime.NumCPU()
	for i := 0; i < core; i++ {
		go parallelMatch(tasks, results)
	}
	wg := sync.WaitGroup{}
	allTaskC := len(siteLinks) * len(keywords)
	go func() {
		c := 0
		for mu := range results {
			c++
			if c%1000 == 0 {
				log.Printf("%d / %d -----> %.2f \n", c, allTaskC, float64(c)/float64(allTaskC))
			}
			if _, ok := km[mu.keyword]; !ok {
				km[mu.keyword] = []*KeywordMatchURL{mu.kum}
			} else {
				km[mu.keyword] = append(km[mu.keyword], mu.kum)
			}
			wg.Done()
		}
	}()

	for _, keyword := range keywords {
		for _, link := range siteLinks {
			wg.Add(1)
			tasks <- parallelTask{
				link:    link,
				keyword: keyword,
			}

		}

	}

	wg.Wait()
	close(tasks)
	close(results)
	return km
}

type parallelTask struct {
	link    *sitethrougher.SiteLinkInfo
	keyword string
}
type result struct {
	keyword string
	kum     *KeywordMatchURL
}

func parallelMatch(tasks <-chan parallelTask, results chan<- result) {
	for task := range tasks {
		results <- result{
			keyword: task.keyword,
			kum:     matchURL(task.link, task.keyword),
		}
	}
}

func matchURL(link *sitethrougher.SiteLinkInfo, keyword string) *KeywordMatchURL {
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
	kmu.HrefTextMatchRate, kmu.HrefTextAllMatch = CalculateMatchRate(keyword, link.HrefTxt)
	return kmu
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
