package goose

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/advancedlogic/goquery"
)

type candidate struct {
	url     string
	width   string
	height  string
	caption string
	surface int
	score   int
}

var largebig = regexp.MustCompile("(large|big)")

var rules = map[*regexp.Regexp]int{
	regexp.MustCompile("(large|big)"):          1,
	regexp.MustCompile("upload"):               1,
	regexp.MustCompile("media"):                1,
	regexp.MustCompile("gravatar.com"):         -1,
	regexp.MustCompile("feeds.feedburner.com"): -1,
	regexp.MustCompile("(?i)icon"):             -1,
	regexp.MustCompile("(?i)logo"):             -1,
	regexp.MustCompile("(?i)spinner"):          -1,
	regexp.MustCompile("(?i)loading"):          -1,
	regexp.MustCompile("(?i)ads"):              -1,
	regexp.MustCompile("badge"):                -1,
	regexp.MustCompile("1x1"):                  -1,
	regexp.MustCompile("pixel"):                -1,
	regexp.MustCompile("thumbnail[s]*"):        -1,
	regexp.MustCompile(".html|" +
		".gif|" +
		".ico|" +
		"button|" +
		"twitter.jpg|" +
		"facebook.jpg|" +
		"ap_buy_photo|" +
		"digg.jpg|" +
		"digg.png|" +
		"delicious.png|" +
		"facebook.png|" +
		"reddit.jpg|" +
		"doubleclick|" +
		"diggthis|" +
		"diggThis|" +
		"adserver|" +
		"/ads/|" +
		"ec.atdmt.com|" +
		"mediaplex.com|" +
		"adsatt|" +
		"view.atdmt"): -1}

func score(tag *goquery.Selection) int {
	src, _ := tag.Attr("src")
	if src == "" {
		src, _ = tag.Attr("data-src")
	}
	if src == "" {
		src, _ = tag.Attr("data-lazy-src")
	}
	if src == "" {
		return -1
	}
	tagScore := 0
	for rule, score := range rules {
		if rule.MatchString(src) {
			tagScore += score
		}
	}

	alt, exists := tag.Attr("alt")
	if exists {
		if strings.Contains(alt, "thumbnail") {
			tagScore--
		}
	}

	id, exists := tag.Attr("id")
	if exists {
		if id == "fbPhotoImage" {
			tagScore++
		}
	}
	return tagScore
}

// WebPageResolver fetches the main image from the HTML page
func WebPageResolver(article *Article) ArticleImage {
	var ret ArticleImage
	doc := article.Doc
	imgs := doc.Find("img")
	var candidates []candidate
	significantSurface := 320 * 200
	significantSurfaceCount := 0
	src := ""
	imgs.Each(func(i int, tag *goquery.Selection) {
		surface := 0
		src, _ = tag.Attr("src")
		if src == "" {
			src, _ = tag.Attr("data-src")
		}
		if src == "" {
			src, _ = tag.Attr("data-lazy-src")
		}
		if src == "" {
			return
		}

		width, _ := tag.Attr("width")
		height, _ := tag.Attr("height")
		alt, _ := tag.Attr("alt")
		if width != "" {
			w, _ := strconv.Atoi(width)
			if height != "" {
				h, _ := strconv.Atoi(height)
				surface = w * h
			} else {
				surface = w
			}
		} else {
			if height != "" {
				surface, _ = strconv.Atoi(height)
			} else {
				surface = 0
			}
		}

		if surface > significantSurface {
			significantSurfaceCount++
		}

		tagscore := score(tag)
		if tagscore >= 0 {
			c := candidate{
				url:     src,
				width:   width,
				height:  height,
				caption: alt,
				surface: surface,
				score:   score(tag),
			}
			candidates = append(candidates, c)
		}
	})

	if len(candidates) == 0 {
		return ret
	}

	if significantSurfaceCount > 0 {
		bestCandidate := findBestCandidateFromSurface(candidates)
		ret.URL = bestCandidate.url
		ret.Width, _ = strconv.Atoi(bestCandidate.width)
		ret.Height, _ = strconv.Atoi(bestCandidate.height)
		ret.Caption = bestCandidate.caption
	} else {
		bestCandidate := findBestCandidateFromScore(candidates)
		ret.URL = bestCandidate.url
		ret.Width, _ = strconv.Atoi(bestCandidate.width)
		ret.Height, _ = strconv.Atoi(bestCandidate.height)
		ret.Caption = bestCandidate.caption
	}

	a, err := url.Parse(ret.URL)
	if err != nil {
		return ret
	}
	finalURL, err := url.Parse(article.FinalURL)
	if err != nil {
		return ret
	}
	b := finalURL.ResolveReference(a)
	ret.URL = b.String()

	return ret
}

func findBestCandidateFromSurface(candidates []candidate) candidate {
	max := 0
	var bestCandidate candidate
	for _, candidate := range candidates {
		surface := candidate.surface
		if surface >= max {
			max = surface
			bestCandidate = candidate
		}
	}

	return bestCandidate
}

func findBestCandidateFromScore(candidates []candidate) candidate {
	max := 0
	var bestCandidate candidate
	for _, candidate := range candidates {
		score := candidate.score
		if score >= max {
			max = score
			bestCandidate = candidate
		}
	}

	return bestCandidate
}

type ogTag struct {
	tpe       string
	attribute string
	name      string
	width     string
	height    string
	caption   string
	value     string
}

var ogTags = [4]ogTag{
	{
		tpe:       "facebook",
		attribute: "property",
		name:      "og:image",
		width:     "og:image:width",
		height:    "og:image:height",
		caption:   "og:description",
		value:     "content",
	},
	// No thumbnails
	// {
	// 	tpe:       "facebook",
	// 	attribute: "rel",
	// 	name:      "image_src",
	// 	value:     "href",
	// },
	{
		tpe:       "twitter",
		attribute: "name",
		name:      "twitter:image",
		width:     "twitter:image:width",
		height:    "twitter:image:height",
		caption:   "twitter:image:alt",
		value:     "value",
	},
	{
		tpe:       "twitter",
		attribute: "name",
		name:      "twitter:image",
		width:     "twitter:image:width",
		height:    "twitter:image:height",
		caption:   "twitter:image:alt",
		value:     "content",
	},
}

type ogImage struct {
	url     string
	width   string
	height  string
	caption string
	tpe     string
	score   int
}

// OpenGraphResolver return OpenGraph properties
func OpenGraphResolver(article *Article) ArticleImage {
	var ret ArticleImage
	var topOgImage ogImage
	doc := article.Doc
	meta := doc.Find("meta")
	links := doc.Find("link")
	meta = meta.Union(links)
	var ogImages []ogImage
	meta.Each(func(i int, tag *goquery.Selection) {
		for _, ogTag := range ogTags {
			attr, exist := tag.Attr(ogTag.attribute)
			value, vexist := tag.Attr(ogTag.value)
			width, _ := tag.Attr(ogTag.width)
			height, _ := tag.Attr(ogTag.height)
			caption, _ := tag.Attr(ogTag.caption)
			if exist && attr == ogTag.name && vexist {
				ogImage := ogImage{
					url:     value,
					width:   width,
					height:  height,
					caption: caption,
					tpe:     ogTag.tpe,
					score:   0,
				}

				ogImages = append(ogImages, ogImage)
			}
		}
	})
	if len(ogImages) == 0 {
		return ret
	}
	if len(ogImages) == 1 {
		ret.URL = ogImages[0].url
		ret.Width, _ = strconv.Atoi(ogImages[0].width)
		ret.Height, _ = strconv.Atoi(ogImages[0].height)
		ret.Caption = ogImages[0].caption
		goto IMAGE_FINALIZE
	}
	for _, ogImage := range ogImages {
		if largebig.MatchString(ogImage.url) {
			ogImage.score++
		}
		if ogImage.tpe == "twitter" {
			ogImage.score++
		}
	}
	topOgImage = findBestImageFromScore(ogImages)
	ret.URL = topOgImage.url
	ret.Width, _ = strconv.Atoi(topOgImage.width)
	ret.Height, _ = strconv.Atoi(topOgImage.height)
	ret.Caption = topOgImage.caption
IMAGE_FINALIZE:
	if !strings.HasPrefix(ret.URL, "http") {
		ret.URL = "http://" + ret.URL
	}

	return ret
}

// assume that len(ogImages)>=2
func findBestImageFromScore(ogImages []ogImage) ogImage {
	max := 0
	var bestOGImage ogImage
	bestOGImage = ogImages[0]
	for _, ogImage := range ogImages[1:] {
		score := ogImage.score
		if score > max {
			max = score
			bestOGImage = ogImage
		}
	}

	return bestOGImage
}
