package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"path"

	"github.com/russross/blackfriday/v2"
)

type BlackFridayRenderer struct {
	blackfriday.HTMLRenderer
	ParentTitle string
	IsIndexFile bool
}

func getAttachmentDir(src string) string {
	u, err := url.Parse(src)
	if err != nil {
		return ""
	}

	if u.Scheme != "" {
		return ""
	}

	dir := path.Dir(src)
	if path.IsAbs(src) {
		return ""
	}

	//如果附件位于assets目录，则提取其上级目录
	if path.Base(dir) == AssetsDirName {
		dir = path.Dir(dir)
	}

	return dir
}

func (r *BlackFridayRenderer) RenderNode(w io.Writer, node *blackfriday.Node, entering bool) blackfriday.WalkStatus {
	//如果是图片，需要检查是否符合图片附件的规则，如果符合，就用Confluence宏而不是HTML输出
	if node.Type == blackfriday.Image {
		dest := string(node.LinkData.Destination)
		dir := getAttachmentDir(dest)
		//预处理图片附件链接
		if dir != "" {
			if entering {
				filename := path.Base(dest)
				result := fmt.Sprintf(`<ac:image><ri:attachment ri:filename="%s">`, filename)

				//同目录下
				if dir == "." {
					//如果不是索引文件，则表示引用的是父页面，需要设置页面名
					//如果是索引文件，则表示引用的是自身页面，不需设置页面名
					if !r.IsIndexFile {
						result += fmt.Sprintf(`<ri:page ri:content-title="%s"/>`, r.ParentTitle)
					}
				} else {
					//不同目录下，则表示其他页面的引用，需要设置页面名
					result += fmt.Sprintf(`<ri:page ri:content-title="%s"/>`, path.Base(dir))
				}

				w.Write([]byte(result))

			} else {
				w.Write([]byte("</ri:attachment></ac:image>"))
			}
			return blackfriday.GoToNext
		}
	}

	//如果是链接，需要检查是否符合附件的规则，如果符合，就用Confluence宏而不是HTML输出
	if node.Type == blackfriday.Link {
		dest := string(node.LinkData.Destination)
		dir := getAttachmentDir(dest)
		//预处理附件链接
		if dir != "" {
			if entering {
				filename := path.Base(dest)
				result := fmt.Sprintf(`<ac:link><ri:attachment ri:filename="%s">`, filename)

				//同目录下
				if dir == "." {
					//如果不是索引文件，则表示引用的是父页面，需要设置页面名
					//如果是索引文件，则表示引用的是自身页面，不需设置页面名
					if !r.IsIndexFile {
						result += fmt.Sprintf(`<ri:page ri:content-title="%s"/>`, r.ParentTitle)
					}
				} else {
					//不同目录下，则表示其他页面的引用，需要设置页面名
					result += fmt.Sprintf(`<ri:page ri:content-title="%s"/>`, path.Base(dir))
				}

				result += "</ri:attachment><ac:plain-text-link-body><![CDATA["
				w.Write([]byte(result))
			} else {
				result := "]]></ac:plain-text-link-body></ac:link>"
				w.Write([]byte(result))
			}
			return blackfriday.GoToNext
		}
	}

	//代码块使用Confluence官方的宏
	//但mermaid代码块除外，因为他需要配合JS渲染成流程图，而不是语法高亮
	if node.Type == blackfriday.CodeBlock && string(node.Info) != "mermaid" {
		result := `<ac:structured-macro ac:name="code">`
		result += `<ac:parameter ac:name="linenumbers">true</ac:parameter>`
		result += `<ac:parameter ac:name="theme">RDark</ac:parameter>`
		result += `<ac:parameter ac:name="language">` + string(node.Info) + `</ac:parameter>`
		result += `<ac:plain-text-body><![CDATA[` + string(node.Literal) + `]]></ac:plain-text-body>`
		result += `</ac:structured-macro>`
		w.Write([]byte(result))
		return blackfriday.GoToNext
	}

	if node.Type == blackfriday.BlockQuote {
		if entering {
			w.Write([]byte(`<ac:structured-macro ac:name="tip">`))
			w.Write([]byte(`<ac:parameter ac:name="title">提示</ac:parameter>`))
			w.Write([]byte(`<ac:rich-text-body>`))
		} else {
			w.Write([]byte(`</ac:rich-text-body>`))
			w.Write([]byte(`</ac:structured-macro>`))
		}
		return blackfriday.GoToNext
	}

	return r.HTMLRenderer.RenderNode(w, node, entering)
}

//Confluence的目录宏，用于自动添加到编译后的页面
const ConfluenceToc = `
<ac:structured-macro ac:name="toc">
	<ac:parameter ac:name="outline">true</ac:parameter>
</ac:structured-macro>
`

func parseMarkdownFile(file, parentTitle string) ([]byte, error) {
	rawData, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	r := &BlackFridayRenderer{
		ParentTitle: parentTitle,
	}
	r.Flags = blackfriday.UseXHTML

	if path.Base(file) == "index.md" {
		r.IsIndexFile = true
	}

	extensions := blackfriday.CommonExtensions
	if EnableHardLineBreak {
		extensions |= blackfriday.HardLineBreak
	}

	mdData := blackfriday.Run(rawData, blackfriday.WithRenderer(r), blackfriday.WithExtensions(extensions))

	return append([]byte(ConfluenceToc), mdData...), nil
}
