package main

import (
	"github.com/mattn/go-gtk/gtk"
	"github.com/mattn/go-gtk/glib"
	"os"
	"github.com/mattn/go-gtk/gdk"
	"fmt"
	"unsafe"
	"github.com/mattn/go-gtk/gdkpixbuf"
	"strconv"
)

type ChessWidget struct {
	Window        *gtk.Window

	BlackImg      *gtk.Image
	WhiteImg      *gtk.Image

	BtnMin        *gtk.Button
	BtnClose      *gtk.Button

	BlackScore    *gtk.Label
	WhiteScore    *gtk.Label
	TimeLab       *gtk.Label

	x, y             int
	startX, startY   int
	w, h             int
	CurrentRole      int    //该谁落子
	TipTimerId       int    //提示闪烁效果定时器
	LeftTimerId      int    //倒计时定时器Id
	machineTimerId   int    //机器落子定时器
	TimeNum          int    //记录剩余时间
	chess [8][8]     int
}

const (
	Empty = iota      //当前棋盘格子没有子
	Black             //当前棋盘格子为黑子
	White             //当前棋盘格子为白子
)

//设置控件
func (c *ChessWidget) CreateWidget() {
	gtk.Init(&os.Args)

	builder := gtk.NewBuilder()
	builder.AddFromFile("G:/GTK/src/黑白棋/黑白棋.glade")

	c.Window = gtk.WindowFromObject(builder.GetObject("window1"))
	c.Window.SetTitle("黑白棋")
	c.Window.SetResizable(false)
	c.Window.SetPosition(gtk.WIN_POS_CENTER)                                       //居中显示
	c.Window.SetDecorated(false)                                            //去边框
	c.Window.SetAppPaintable(true)                                          //允许绘图
	c.Window.SetEvents(int(gdk.BUTTON_PRESS_MASK | gdk.BUTTON1_MOTION_MASK))       //允许鼠标键盘事件

	c.BtnMin = gtk.ButtonFromObject(builder.GetObject("btnMin"))
	c.BtnClose = gtk.ButtonFromObject(builder.GetObject("btnClose"))
	ButtonSetImageFromFile(c.BtnMin, "G:/GTK/src/黑白棋/images/available_pos.png")
	ButtonSetImageFromFile(c.BtnClose, "G:/GTK/src/黑白棋/images/available_pos.png")

	c.BlackScore = gtk.LabelFromObject(builder.GetObject("BlackScore"))
	c.WhiteScore = gtk.LabelFromObject(builder.GetObject("WhiteScore"))
	c.TimeLab = gtk.LabelFromObject(builder.GetObject("TimeLab"))
	c.BlackScore.ModifyFontSize(30)
	c.WhiteScore.ModifyFontSize(30)
	c.WhiteScore.ModifyFG(gtk.STATE_NORMAL, gdk.NewColor("white"))
	c.TimeLab.ModifyFontSize(30)
	c.BlackScore.SetText("2")
	c.WhiteScore.SetText("2")
	//c.TimeLab.SetText("20")


	c.BlackImg = gtk.ImageFromObject(builder.GetObject("BlackImg"))
	c.WhiteImg = gtk.ImageFromObject(builder.GetObject("WhiteImg"))
	ImageSetPicFromFile(c.BlackImg, "G:/GTK/src/黑白棋/images/black.png")
	ImageSetPicFromFile(c.WhiteImg, "G:/GTK/src/黑白棋/images/white.png")

	c.startX, c.startY = 286, 186
	c.w, c.h = 47, 47

	c.Window.ShowAll()
}

//给image设置图片
func ImageSetPicFromFile(img *gtk.Image, filename string) {
	var w, h int
	img.GetSizeRequest(&w, &h)
	pixbuf, _ := gdkpixbuf.NewPixbufFromFileAtScale(filename, w, h, false)
	img.SetFromPixbuf(pixbuf)
	pixbuf.Unref()
}

//给按钮设置图标
func ButtonSetImageFromFile(btn *gtk.Button, filename string) {
	var w, h int
	btn.GetSizeRequest(&w, &h)
	pixbuf, _ := gdkpixbuf.NewPixbufFromFileAtScale(filename, w, h, false)
	image := gtk.NewImageFromPixbuf(pixbuf)
	pixbuf.Unref()
	btn.SetImage(image)
	btn.SetCanFocus(false)            //去掉按钮焦距
}

func (c *ChessWidget) JudgeResult() {
	//判断胜负条件,双方都不能吃子
	isOver := true //默认游戏结束

	blackNum, whiteNum := 0, 0
	for i:=0; i<8; i++ {
		for j:=0; j<8; j++ {
			if c.chess[i][j] == Black {
				blackNum++
			} else if c.chess[i][j] == White {
				whiteNum++
			}

			if c.JudgeRule(i, j, Black, false) > 0 || c.JudgeRule(i, j, White, false) > 0 {
				isOver = false
			}
		}
	}
	fmt.Println(isOver)
	//界面显示个数
	c.BlackScore.SetText(strconv.Itoa(blackNum))
	c.WhiteScore.SetText(strconv.Itoa(whiteNum))

	if isOver == false {
		return
	}

	//执行到这,说明游戏结束
	glib.TimeoutRemove(c.TipTimerId)
	glib.TimeoutRemove(c.LeftTimerId)

	//判断胜负
	var result string
	if blackNum < whiteNum {
		result = "白棋胜利\n是否继续游戏"
	} else if blackNum > whiteNum {
		result = "黑棋胜利\n是否继续游戏"
	} else {
		result = "平局\n是否继续游戏"
	}

	//问题对话框
	dialog := gtk.NewMessageDialog(
		c.Window,                  //父窗口
		gtk.DIALOG_MODAL,          //模态对话框
		gtk.MESSAGE_QUESTION,      //问题对话框
		gtk.BUTTONS_YES_NO,        //按钮
	    result)                    //对话框内容
	ret := dialog.Run()
	if ret == gtk.RESPONSE_YES {   //按了继续游戏
		c.InitChess()              //重新初始化棋盘
	} else {                       //关闭窗口
		gtk.MainQuit()
	}
	dialog.Destroy()
}

func (c *ChessWidget) MachinePlay() {
	//移除定时器
	glib.TimeoutRemove(c.machineTimerId)

	max, px, py := 0, -1, -1

	//优先落子在4个角落,如果4个角落不能吃子,则选择最多的
	for i:=0; i<8; i++ {
		for j := 0; j < 8; j++ {
			//最后一个参数为false,只判断能否吃子,不改变二维数组
			num := c.JudgeRule(i, j, c.CurrentRole, false)
			if num > 0 {
				if (i == 0 && j == 0) || (i == 7 && j == 0) || (i == 0 && j == 7) || (i == 7 && j == 7) {
					px, py = i, j
					goto End
				}

				if num  > max {
					max, px, py = num, i, j
				}
			}
		}
	}
End:
	if px == -1 {//说明机器不能落子
		c.ChangeRole()
		return
	}

	//机器吃子
	c.JudgeRule(px, py, c.CurrentRole, true)
	//刷新绘图区域(整个窗口)
	c.Window.QueueDraw()
	//改变角色
	c.ChangeRole()
}

func (c *ChessWidget) ChangeRole() {
	//重新设置时间
	c.TimeNum = 20
	c.TimeLab.SetText(strconv.Itoa(c.TimeNum))

	c.BlackImg.Hide()
	c.WhiteImg.Hide()
	if c.CurrentRole == Black {
		c.CurrentRole = White
	} else {
		c.CurrentRole = Black
	}

	c.JudgeResult() //统计个数,判断胜负

	if c.CurrentRole == White {
		c.machineTimerId = glib.TimeoutAdd(1000, func() bool {
			c.MachinePlay() //机器落子
			return  true
		})
	}
}

//鼠标点击事件函数
func MousePressEvent(ctx *glib.CallbackContext) {
	//获取用户传递的参数
	data := ctx.Data()
	c, ok := data.(*ChessWidget)
	if !ok {
		fmt.Println("MousePressEvent err", c)
		return
	}
	//获取鼠键按下树形结构体变量,系统内部的变量,不是用户传参变量
	arg := ctx.Args(0)
	event := *(**gdk.EventButton)(unsafe.Pointer(&arg))
	c.x, c.y = int(event.X), int(event.Y)
	i, j := (c.x - c.startX)/c.w, (c.y - c.startY)/c.h

	if c.CurrentRole == White {//如果是白棋下,就是机器,用户不能点击
	    return
	}

	if i >= 0 && i <= 7 && j >= 0 && j <= 7 {
		//c.chess[i][j] = c.CurrentRole
		//吃子,落子必须要能吃子
		if c.JudgeRule(i, j, c.CurrentRole, true) > 0 {
			//刷新绘图区域(整个窗口)
			c.Window.QueueDraw()
			//改变角色
			c.ChangeRole()
		}
	}
}

//鼠标移动事件
func MouseMoveEvent(ctx *glib.CallbackContext) {
	//获取用户传递的参数
	data := ctx.Data()
	c, ok := data.(*ChessWidget)
	if !ok {
		fmt.Println("MouseMoveEvent err", c)
		return
	}
	//获取鼠键按下树形结构体变量,系统内部的变量,不是用户传参变量
	arg := ctx.Args(0)
	event := *(**gdk.EventButton)(unsafe.Pointer(&arg))
	mx, my := int(event.XRoot) - c.x, int(event.YRoot) - c.y
	c.Window.Move(mx, my)
}

func DrawWindowImageFromFile(ctx *glib.CallbackContext) {
	//获取用户传递的参数
	data := ctx.Data()
	c, ok := data.(*ChessWidget)
	if !ok {
		fmt.Println("MouseMoveEvent err", c)
		return
	}
	//获取画家,设置绘图区域
	path := "G:/GTK/src/黑白棋/images/bg.jpg"
	boardpath := "G:/GTK/src/黑白棋/images/board.jpg"
	blackpath := "G:/GTK/src/黑白棋/images/black.png"
	whitepath := "G:/GTK/src/黑白棋/images/white.png"
	painter := c.Window.GetWindow().GetDrawable()
	gc := gdk.NewGC(painter)

	//新建pixbuf
	pixbuf, _ := gdkpixbuf.NewPixbufFromFileAtScale(path, 1014, 730, false)
	boardpixbuf, _ := gdkpixbuf.NewPixbufFromFileAtScale(boardpath, 450, 450, false)
	//黑白棋pixbuf
	blackpixbuf, _ := gdkpixbuf.NewPixbufFromFileAtScale(blackpath, c.w, c.h, false)
	whitepixbuf, _ := gdkpixbuf.NewPixbufFromFileAtScale(whitepath, c.w, c.h, false)

	//画图
	painter.DrawPixbuf(gc, pixbuf, 0, 0, 0, 0,
		-1, -1, gdk.RGB_DITHER_NONE, 0, 0)
	painter.DrawPixbuf(gc, boardpixbuf, 0, 0, 250, 150,
		-1, -1, gdk.RGB_DITHER_NONE, 0, 0)

	//画黑白棋
	for i:= 0; i<8 ; i++ {
		for j:=0; j<8; j++ {
			if c.chess[i][j] == Black {
				painter.DrawPixbuf(gc, blackpixbuf, 0, 0, c.startX + i * c.w, c.startY + j * c.h,
					-1, -1, gdk.RGB_DITHER_NONE, 0, 0)
			} else if c.chess[i][j] == White {
				painter.DrawPixbuf(gc, whitepixbuf, 0, 0, c.startX + i * c.w, c.startY + j * c.h,
					-1, -1, gdk.RGB_DITHER_NONE, 0, 0)
			}
		}
	}

	//释放资源
	pixbuf.Unref()
	blackpixbuf.Unref()
	whitepixbuf.Unref()
}

//事件,信号处理
func (c *ChessWidget) HandleSignal() {
	c.Window.Connect("button-press-event", MousePressEvent, c)      //鼠标点击事件
	c.Window.Connect("motion-notify-event", MouseMoveEvent, c)      //鼠标移动事件
	c.BtnMin.Clicked(func() {                                         //按钮最小化事件
		c.Window.Iconify()
	})
	c.BtnClose.Clicked(func(){                                        //按钮关闭事件
	    glib.TimeoutRemove(c.TipTimerId)                              //关闭定时器
	    glib.TimeoutRemove(c.LeftTimerId)
		gtk.MainQuit()
	})
	c.Window.Connect("configure_event", func() {                    //窗口改变事件
		c.Window.QueueDraw()                                          //重新刷图
	})
	c.Window.Connect("expose-event", DrawWindowImageFromFile, c)    //绘图事件
}

//提示功能,实现闪烁效果
func ShowTip(c *ChessWidget) {
	if c.CurrentRole == Black {
		c.WhiteImg.Hide()
		if c.BlackImg.GetVisible() == true {
			c.BlackImg.Hide()
		} else {
			c.BlackImg.Show()
		}
	} else {
		c.BlackImg.Hide()
		if c.WhiteImg.GetVisible() == true {
			c.WhiteImg.Hide()
		} else {
			c.WhiteImg.Show()
		}
	}
}

//黑白棋属性相关
func (c *ChessWidget) InitChess() {
	//初始化棋盘
	for i:=0; i<8; i++ {
		for j:=0; j<8; j++ {
			c.chess[i][j] = Empty
		}
	}
	c.chess[3][3] = Black
	c.chess[4][4] = Black
	c.chess[3][4] = White
	c.chess[4][3] = White

	//更新一下棋盘
	c.Window.QueueDraw()
	c.BlackScore.SetText("2")
	c.WhiteScore.SetText("2")

	//image都先隐藏
	c.WhiteImg.Hide()
	c.BlackImg.Hide()

	//默认黑棋先下
	c.CurrentRole = Black

	//启动定时器
	c.TipTimerId = glib.TimeoutAdd(500, func() bool {
		ShowTip(c)
		return true
	})

	//倒计时定时器
	c.TimeNum = 20
	c.TimeLab.SetText(strconv.Itoa(c.TimeNum))
	c.LeftTimerId = glib.TimeoutAdd(1000, func() bool {
		c.TimeNum--
		c.TimeLab.SetText(strconv.Itoa(c.TimeNum))
		if c.TimeNum == 0 {
			c.ChangeRole()
		}
		return true
	})
}

func (c *ChessWidget) JudgeRule(x, y int, role int, eatChess bool) (eatNum int) {
	//棋盘的八个方向
	dir := [8][2]int{{1, 0}, {1, -1}, {0, -1}, {-1, -1}, {-1, 0}, {-1, 1}, {0, 1}, {1, 1}}
	tmpX, tmpY := x, y                       //临时保存棋盘数组目标位置
	if c.chess[tmpX][tmpY] != Empty {        //如果此方格内有棋子,则返回
		return 0
	}

	for i:=0; i<8; i++ {
		tmpX += dir[i][0]
		tmpY += dir[i][1]
		if (tmpX < 8 && tmpX >= 0 && tmpY < 8 && tmpY >= 0) &&
			(c.chess[tmpX][tmpY] != role) && (c.chess[tmpX][tmpY] != Empty) {
			for tmpX < 8 && tmpX >= 0 && tmpY < 8 && tmpY >= 0 {
				if c.chess[tmpX][tmpY] == Empty {
					break
				}

				if c.chess[tmpX][tmpY] == role {//找到自己棋子,代表可以吃子
					if eatChess == true {       //确定吃子
						c.chess[x][y] = role    //开始点标志位自己的棋子
						tmpX -= dir[i][0]
						tmpY -= dir[i][1]        //后退一步
						for (tmpX != x) || (tmpY != y) {
							c.chess[tmpX][tmpY] = role   //标志位自己的棋子
							tmpX -= dir[i][0]
							tmpY -= dir[i][1]            //继续后退一步
							eatNum++                     //累计
						}
					} else {//不吃子,只是判断这个位置能不能吃子
						tmpX -= dir[i][0]
						tmpY -= dir[i][1]                 //后退一步
						for (tmpX != x) || (tmpY != y) {
							tmpX -= dir[i][0]
							tmpY -= dir[i][1]             //继续后退一步
							eatNum++                      //累计
						}
					}
					break
				}//没有找到自己的棋子,就向前走一步
				tmpX += dir[i][0]
				tmpY += dir[i][1]
			}
		}//如果这个方向不能吃子,就换一个方向
		tmpX, tmpY = x, y
	}
	return
}


func main() {
	var obj ChessWidget
	obj.CreateWidget()
	obj.HandleSignal()
	obj.InitChess()

	gtk.Main()
}