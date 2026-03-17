let enbPlatform = '', enbPlatformType = '';
//颜色值转换  十六进制转换为rgb 
function changeColor(col){
	
	let newStr = (col.toLowerCase()).replace(/\#/g,'');
	let len = newStr.length;
	
	if(len == 3){
		let t = '';
		for (let i=0;i<len;i++){
			t += newStr.slice(i,i+1).concat(newStr.slice(i,i+1))
		}
		newStr = t;
	}
	
	let arr = [];
	for(let i=0;i<6;i=i+2){
		let s = newStr.slice(i,i+2)
		arr.push(parseInt("0x"+s))
	}
	return arr.join(",")
}

function createRoundedFavicon(color = '#FF4614', radius = 10) {
  // 1. 创建Canvas画布（推荐32x32尺寸）
  const canvas = document.createElement('canvas');
  const size = 32; // 标准favicon尺寸
  canvas.width = size;
  canvas.height = size;
  const ctx = canvas.getContext('2d');
  
  // 2. 绘制圆角矩形
  ctx.clearRect(0, 0, size, size);
  ctx.fillStyle = color;
  
  // 圆角矩形路径算法
  ctx.beginPath();
  ctx.moveTo(radius, 0);
  ctx.lineTo(size - radius, 0);
  ctx.arcTo(size, 0, size, radius, radius);
  ctx.lineTo(size, size);
  ctx.arcTo(size, size, size, size, 0);
  ctx.lineTo(radius, size);
  ctx.arcTo(0, size, 0, size - radius, radius);
  ctx.lineTo(0, 0);
  ctx.arcTo(0, 0, 0, 0, 0);
  ctx.closePath();
  ctx.fill();
  
  // 3. 转换为Data URL
  return canvas.toDataURL('image/png');
}

// 4. 更新页面favicon
function updateFavicon(color) {
  const faviconUrl = createRoundedFavicon(color);
  let link = document.querySelector("link[rel*='icon']");
  
  if (!link) {
    link = document.createElement('link');
    link.rel = 'icon';
    document.head.appendChild(link);
  }
  
  link.href = faviconUrl;
  link.type = 'image/png';
}

Array.prototype.remove = function(item) {
	var idx = this.indexOf(item);
	
	if(idx >= 0) this.splice(idx, 1);
}

$.ajaxSetup({
	beforeSend: function(){
		var paramstr = arguments[1].data,bool=true;
		if(paramstr){
			var paramArr = paramstr.split('&');
			paramArr.map(function(item){
				var codes = item.split('=');
				if(!validXSS(codes[1])) bool = false;
			});
		}
		if(!bool) {
			try{
				$('.messager-window:contains(tips)').panel('destroy');
			}catch(e){}
			toast(XSSValidInfo,'#omc_app_ctn');
			setTimeout(function(){
				$('.datagrid-mask,.datagrid-mask-msg').hide();
			},1500);
		}
		return bool;
	}
});
function elementDialogLoadByUrl(url, el, vm){
	var params = {};
	if(vm.params && typeof vm.params == 'object') {
		Object.assign(params,vm.params);
	}
	el.innerHTML = '';
	$.ajax({
        type: 'post',
        url: url,
        data: params,
        dataType: 'html',
        success: function(html) {
          el.innerHTML = html;
          var scripts = el.querySelectorAll('script');
          setTimeout(function() {
            Array.from(scripts).map(function(script) { /* 执行远程的脚本 */
              if (el.contains(script)) {
            	  el.removeChild(script);
              }
              var newScript = document.createElement('script');
              newScript.type = 'text/javascript';
              newScript.innerHTML = script.innerHTML;
              el.appendChild(newScript);
            });
            vm.$emit('success');
          }, 0);
          
          setTimeout(function() {
              try{
            	  $.parser.parse(el);
              }catch(e){}
          }, 0);
        },
        error: function() {
          
        }
      });
}
/* 扩展Vue获取实例方法 */
Vue.getInstance = function(el) {
	return document.querySelector(el).__vue__;
}
/* 检查是否含XSS脚本 */
function validXSS(val){
	if(filterSpecialCharactersEnable != '1') return true;
	
	var words = ['script','iframe','onclick','onfocus','onerror','onchange'],bool=true;
	words.map(function(item){
		if(val && (val+'').toLowerCase().indexOf(item)>=0) bool = false;
	});
	return bool;
}
function checkForm(form){
	if(filterSpecialCharactersEnable != '1') return true;
	
	var bool = true;
	try{
		$(':input',form).each(function(n,item){
			var namedItem = form[item.name];
			if(namedItem){
				if(!validXSS(namedItem.value)) bool = false;
			}
		})
	}catch(e){}
	if(!bool) {
		try{
			$('.messager-window:contains(tips)').panel('destroy');
		}catch(e){}
		toast(XSSValidInfo,'#omc_app_ctn');
		setTimeout(function(){
			$('.datagrid-mask,.datagrid-mask-msg').hide();
		},1500);
	}
	return bool;
}
function checkParams(param){
	var bool = true;
	for(var key in param){
		if(!validXSS(param[key])) bool = false;
	}
	if(!bool) {
		try{
			$('.messager-window:contains(tips)').panel('destroy');
		}catch(e){}
		toast(XSSValidInfo,'#omc_app_ctn');
		setTimeout(function(){
			$('.datagrid-mask,.datagrid-mask-msg').hide();
		},1500);
	}
	return bool;
}
$.parser.onComplete = layoutOnComplete;

//点击tab页切换内容
function turnTabs(ele){
	var tabClass = $(ele).attr('tabtit');
	var thisPar = $(ele).closest('.tabsTitle').parent();
	
	$(ele).siblings().removeClass("active");  
	$(ele).addClass("active");
	$("." + tabClass).show().siblings("div").hide();
	$(window).resize();
	
	if(thisPar.find("div").hasClass('omcTabsPage_second')){
		$("." + tabClass).find(".omcPageTitleContainer_second > li").first().click();
	}	
	$('table.datagrid-f').datagrid('resize');
}

function layoutOnComplete (ctx) {
	// 去掉单选表格中的全选按钮
	try {
		$("table.datagrid-f").each(function() {
			var id = $(this).attr("id");
			var singleSelect = $("#" + id).datagrid("options")["singleSelect"];
			if (singleSelect) {
				$("#" + id).siblings("div.datagrid-view2").addClass("singleSelectGrid");
			}
		});
		initInputs(ctx);
	} catch (e) {
		console.info(e);
	}
}

// 创建页面左侧二级菜单
function createSecondMenu(data, nodeClickFunc, container) {
	container.addClass("secondMenuContainer");
	var node = $("<li></li>");
	for (var i = 0; i < data.length; i++) {
		if (data[i] == "-") {
			container.append($("<hr/>"));
			continue;
		}
		
		var node = $("<li>" + data[i]["text"] + "</li>");
		node.attr("url", data[i]["url"]);
		node.bind("click", function() {
			$(this).siblings("li").removeClass("tree-node-selected");
			$(this).addClass("tree-node-selected");
			nodeClickFunc($(this));
		});
		if (i == 0) {
			node.addClass("tree-node-selected");
		}
		container.append(node);
	}
	
}

// 关闭遮罩
function closeLoading(parent_ele) {
	var load_ele = "";
	if (parent_ele) {
		load_ele = "#" + parent_ele + " #Loading";
	} else {
		load_ele = "#Loading";
	}
	$(load_ele).hide("fast", function() {
				$(this).remove();
			});
}


//CPE历史，单元格样式
function cpeHisColorStyler(value, rowData, rowIndex) {
	return "background: #E1F3Fb !important";
}

//CPE要显示的历史数据，单元格样式
function cpeDataColorStyler(value, rowData, rowIndex) {
	return "background: #F2F8FE !important";
}

function alarmStyler(value, rowData, rowIndex) {
	if (value == null) {
		return null;
	} else if (value == "Critical") {
		return "background: #FF7B7B !important";
	} else if (value == "Major") {
		return "background: #FB9F50 !important";
	} else if (value == "Minor") {
		return "background: #D0D53B !important";
	} else {
		return "background: #67DFF8 !important";
	}
}

//给javascript 的日期类型添加一个格式化方法
Date.prototype.Format = function (fmt) { //author: meizz
    var o = {
        "M+": this.getMonth() + 1,                 //月份
        "d+": this.getDate(),                    //日
        "h+": this.getHours(),                   //小时
        "m+": this.getMinutes(),                 //分
        "s+": this.getSeconds(),                 //秒
        "q+": Math.floor((this.getMonth() + 3) / 3), //季度
        "S": this.getMilliseconds()             //毫秒
    };
    if (/(y+)/.test(fmt))
        fmt = fmt.replace(RegExp.$1, (this.getFullYear() + "").substr(4 - RegExp.$1.length));
    for (var k in o)
        if (new RegExp("(" + k + ")").test(fmt))
            fmt = fmt.replace(RegExp.$1, (RegExp.$1.length == 1) ? (o[k]) : (("00" + o[k]).substr(("" + o[k]).length)));
    return fmt;
};

String.prototype.trim = function() {
    return this.replace(/(^\s*)|(\s*$)/g, '');
}


/**
 * 数据表格中，列的格式化函数，作用:判断PCI状态，是否锁定
 * @param value 字段的值
 * @param rowData 行的数据
 * @param rowIndex 行的索引
 */


function pciStatus(value, rowData, rowIndex){
	var reg = new RegExp("^(IDU\/CN)");
	if("LTE WiFi VoIP Gateway" == rowData["OLDPRODUCT"] || (reg.test(rowData["OLDPRODUCT"])==true) ){
		return "--";
	}
	if(!value) return "";
	var lock_pci = new Array();
	lock_pci = value.split("_");
	var pciValue = lock_pci[1];
	if(lock_pci[0] == 1){
		var imgL = "<span class='el-icon el-icon-operation-lock'></span><span>"+pciValue+"</span>" ;
		return imgL;
	}else{
		var imgL = "<span style='font-size:14px' class='el-icon el-icon-status-unlock'></span><span>"+pciValue+"</span>";
		return imgL;
	}
}

/**
 * 数据表格中，列的格式化函数，作用是：给每个Cell添加鼠标悬停提示框
 * @param value 字段的值
 * @param rowData 行的数据
 * @param rowIndex 行的索引
 */
function gridCellTooltipFormatter(value, rowData, rowIndex){
	if (!value) return "";
	var retStr = "<span class='jbox_tooltip' title='" + value + "'>" + value + "</span>";
	return retStr;
}
function gridCellTooltiponLoadSuccess(){
	$(this).datagrid("fixRownumber");
	$(this).datagrid("enableContextmenuAutoSize");
	
	$(".jbox_tooltip").jBox('Tooltip',{
		pointer: false,
		color: 'black',
		animation: {open: 'slide:right', close: 'slide:right'}
	});
}

(function($) {
	$.fn.serializeJson = function() {
		var serializeObj = {};
		var array = this.serializeArray();// 将表单序列化成数组
		$(array).each(
			function() {
				if (serializeObj[this.name]) {
					if ($.isArray(serializeObj[this.name])) {
						serializeObj[this.name].push(this.value);
					} else {
						serializeObj[this.name] = [serializeObj[this.name], this.value ];
					}
				} else {
					serializeObj[this.name] = this.value;
				}
			});
		return serializeObj;
	};
})(jQuery);

/**
 * 定位到HTML的元素
 * @param {} jq jquery对象
 * @param {} _185 要定位的DOM对象
 * @return {}
 */
function scrollTo(jq, _185) {
	if(typeof jq == 'object'){
		return jq.each(function() {
			Positioning(this, _185);
		});
	}
}

function Positioning(_12e, _12f) {
	var c = $(_12e).parent();
	while (c[0].tagName != "BODY" && c.css("overflow-y") != "auto") {
		c = c.parent();
	}
	var n = $(_12f);
	var ntop = n.offset().top;
	if (c[0].tagName != "BODY") {
		var ctop = c.offset().top;
		if (ntop < ctop) {
			c.scrollTop(c.scrollTop() + ntop - ctop);
		} else {
			if (ntop + n.outerHeight() > ctop + c.outerHeight() - 18) {
				c.scrollTop(c.scrollTop() + ntop + n.outerHeight() - ctop
						- c.outerHeight() + 18);
			}
		}
	} else {
		c.scrollTop(ntop);
	}
}


/**
 * 刷新表格
 * @param datagridId 表格的ID
 * @param params 查询条件
 */
function doSearch(datagridId, params) {
	params.timeZone=timeZone;
	$('#' + datagridId).datagrid({
		queryParams : params,
		pageNumber : 1
	});
}

function doSearchUrl(datagridId, params,url) {
	params.timeZone=timeZone;
	$('#' + datagridId).datagrid({
		queryParams : params,
		pageNumber : 1,
		url:url
	});
}


function logout(ctx) {
	var params = {},
		path = ctx? ctx+'/':'/',
		root = ctx? ctx:'/';
	$.ajax({
		url: path + 'sys/login/logout2load.htm',
	    type: 'POST',
	    dataType: 'text',
	    data :params,
	    contentType: "application/x-www-form-urlencoded; charset=utf-8",
	    error: function(request){
        if ( window.opener ) {
          window.close();			
            window.opener.location = root;
        } else {
          window.location = root;
        }
	    },
	    success: function(request){
        localStorage.setItem("keyData",'');
	    	if ( window.opener ) {
          window.close();			
            window.opener.location = root;
        } else {
          window.location = root;
        }
		   
	    }
	});
}

/**
 * 判断输入的值是否是大于0的数字
 * @param {} obj
 */
function validateNumber(obj){
	if(obj.value != null && obj.value != ''){
		if(isNaN(obj.value)){
			obj.value = '';
		}else{
			var vl = parseFloat(obj.value);
			if(vl <= 0){
				obj.value = '';
			}
		}
	}
}
/*<%-- 验证，必填项，无其它限制条件 --%>*/
function validateRequired(e){
	var ele = $(e["target"]);
    var currValLength = ele.val().length;
    if (currValLength == 0) {
		  //$("#" + ele.attr("id") + "_err").show();
      $("#" + ele.attr("id") + "_err").addClass('redColor');
		  ele.addClass("err_border");
    } else {
    	//$("#" + ele.attr("id") + "_err").hide();
      $("#" + ele.attr("id") + "_err").removeClass('redColor');
    	ele.removeClass("err_border");
    }
}

/*<%--验证最大长度和最小长度--%>*/
function validateMaxAndMinLength(e) {
    var ele = $(e["target"]);
    var must = ele.attr("must");
    var maxLength = ele.attr("max_length");
    var minLength = ele.attr("min_length");
    var currValLength = ele.val().length;
    if (currValLength == 0) {
    	if (must == 1) {
    		//$("#" + ele.attr("id") + "_err").show();
        $("#" + ele.attr("id") + "_err").addClass('redColor');
    		ele.addClass("err_border");
    	} else {
    		//$("#" + ele.attr("id") + "_err").hide();
        $("#" + ele.attr("id") + "_err").removeClass('redColor');
    		ele.removeClass("err_border");
    	}
    	return;
    }
    
    var errShowFlag = false;

    if(ele.val().indexOf(';') >= 0) {
      errShowFlag = true;
    }

    if (maxLength) {// 大于最大值
    	if (currValLength > maxLength) {
    		errShowFlag = true;
    	}
    }
    if (minLength) {
    	if (currValLength < minLength) {// 小于最小值
    		errShowFlag = true;
    	}
    }
    
    if (errShowFlag) {
        //$("#" + ele.attr("id") + "_err").show();
        $("#" + ele.attr("id") + "_err").addClass('redColor');
        ele.addClass("err_border");
        throw(new Error('Invalid'));
    } else {
        //$("#" + ele.attr("id") + "_err").hide();
        $("#" + ele.attr("id") + "_err").removeClass('redColor');
        ele.removeClass("err_border");
    }
}

/*<%--验证范围20..30中的20和30的大小关系--%>*/
function validateRangeNumber(e) {
    var ele = $(e["target"]);
    var must = ele.attr("must");
    var paramValue = ele.val();
    var currValLength = ele.val().length;
    if (currValLength == 0) {
    	if (must == 1) {
    		//$("#" + ele.attr("id") + "_err").show();
        $("#" + ele.attr("id") + "_err").addClass('redColor');
    		ele.addClass("err_border");
    	} else {
    		//$("#" + ele.attr("id") + "_err").hide();
        $("#" + ele.attr("id") + "_err").removeClass('redColor');
    		ele.removeClass("err_border");
    	}
    	return;
    }
    if (paramValue.indexOf("..")>-1) {
    	var errShowFlag = false;
    	var rangeNumberList = paramValue.split("..");
    	if (rangeNumberList.length != 2) {
    		 errShowFlag = true; 
    	} else {
    		if (rangeNumberList[0] > rangeNumberList[1]) {
    			errShowFlag = true;
    		}
    	}
	    if (errShowFlag) {
	        //$("#" + ele.attr("id") + "_err").show();
          $("#" + ele.attr("id") + "_err").addClass('redColor');
	        ele.addClass("err_border");
	    } else {
	        //$("#" + ele.attr("id") + "_err").hide();
          $("#" + ele.attr("id") + "_err").removeClass('redColor');
	        ele.removeClass("err_border");
	    }
    } else {
    	return;
    }
}
// 根据正则表达式验证并判断大小
function validateMaxAndMinVal_float(e) {
	var ele = $(e["target"]);
    var reg = /^\d+(\.\d)?$/;
    var currVal = ele.val();
    var must = ele.attr("must");
    var minValue = ele.attr("min_value")-0;
    var maxValue = ele.attr("max_value")-0;
    
    var currValLength = currVal.length;
    if (currValLength == 0) {
    	if (must == 1) {
    		//$("#" + ele.attr("id") + "_err").show();
        $("#" + ele.attr("id") + "_err").addClass('redColor');
    		ele.addClass("err_border");
    	} else {
    		//$("#" + ele.attr("id") + "_err").hide();
        $("#" + ele.attr("id") + "_err").removeClass('redColor');
    		ele.removeClass("err_border");
    	}
    	return;
    }
    
    var showFlag = false;
    if (!reg.test(currVal)) {
        showFlag = true;
    } else {
    	currVal = currVal-0;
    	  if (ele.attr("min_value")) {
        	if (currVal < minValue) {
            	showFlag = true;
        	}
    		}
    		if (ele.attr("max_value")) {
        	if (currVal > maxValue) {
            	showFlag =true;
        	}
    	}
    }
    if (showFlag) {
        //$("#" + ele.attr("id") + "_err").show();
        $("#" + ele.attr("id") + "_err").addClass('redColor');
        ele.addClass("err_border");
    } else {
        //$("#" + ele.attr("id") + "_err").hide();
        $("#" + ele.attr("id") + "_err").removeClass('redColor');
        ele.removeClass("err_border");
    }
}
/*<%--验证是否为数字，是否超出最大值，是否超出最小值--%>*/
function validateMaxAndMinVal(e) {
    var ele = $(e["target"]);
    var reg = /^-?\d*$/;
    reg = /^(-?[1-9]\d*|0)$/;
    var currVal = ele.val();
    var must = ele.attr("must");
    var currValLength = currVal.length;
    if (currValLength == 0) {
    	if (must == 1) {
    		//$("#" + ele.attr("id") + "_err").show();
        $("#" + ele.attr("id") + "_err").addClass('redColor');
    		ele.addClass("err_border");
    	} else {
    		//$("#" + ele.attr("id") + "_err").hide();
        $("#" + ele.attr("id") + "_err").removeClass('redColor');
    		ele.removeClass("err_border");
    	}
    	return;
    }
    /*<%--先判断是否为整型--%>*/
    if (reg.test(currVal) && currVal != '-0') {
    	currVal = parseInt(currVal);
    } else {
    /*<%--不是整型，显示错误，返回--%>*/
        //$("#" + ele.attr("id") + "_err").show();
        $("#" + ele.attr("id") + "_err").addClass('redColor');
        ele.addClass("err_border");
        return;
    }
    /*<%--再判断取值范围是否合法--%>*/
    var showFlag = false;
    if (ele.attr("min_value")) {
    /*<%--有最小值--%>*/
        var minValue = parseInt(ele.attr("min_value"), 10);
        if (currVal < minValue) {
            showFlag = true;
        }
    }
    if (ele.attr("max_value")) {
    /*<%--有最大值--%>*/
        var maxValue = parseInt(ele.attr("max_value"), 10);

        if(ele.attr("max_value").length > 16) {
          showFlag = !isLessThanBignum(ele);
        }else {
          if (currVal > maxValue) {
              showFlag =true;
          }
        }
    }
    if (showFlag) {
        //$("#" + ele.attr("id") + "_err").show();
        $("#" + ele.attr("id") + "_err").addClass('redColor');
        ele.addClass("err_border");
    } else {
        //$("#" + ele.attr("id") + "_err").hide();
        $("#" + ele.attr("id") + "_err").removeClass('redColor');
        ele.removeClass("err_border");
    }
}

function isLessThanBignum(el) {
    var range = el.attr('title').split('-'),
        max = range[1],
        num = el.val()+'',
        bool = true,
        list = [];
    
    if(num.length > max.length || isNaN(num)) bool = false;

    if(num.length == max.length) {
        for(var i = 0; i < max.length; i++) {
            var isBig = max[i] - num[i] >= 0;

            list.push(max[i] - num[i] > 0);

            if(!isBig) {
                var some = list.filter(function(item){ return item == true;});

                if(some.length == 0) {
                    bool = false;
                    break;
                }
            }
        }
    }

    return bool;
}

/*<%--验证是否为数字，是否","分割  是否超出最大值，是否超出最小值--%>*/
function validateMaxAndMinValSplit(e) {
    var ele = $(e["target"]);
    var reg = /^-*\d*$/;
    reg = /^-?\d+$/;
    var currVal = ele.val();//比较的值
    if(currVal[(currVal.length-1)] == ","){
    	currVal = currVal.substring(0,currVal.length-1);
    	//$("#" + ele.attr("id") + "_err").show();
      $("#" + ele.attr("id") + "_err").addClass('redColor');
		  ele.addClass("err_border");
		  return;
    }
    var currValArr = currVal.split(",");
    var must = ele.attr("must");
    var currValLength = currVal.length;
    if (currValLength == 0) {
    	if (must == 1) {
    		//$("#" + ele.attr("id") + "_err").show();
        $("#" + ele.attr("id") + "_err").addClass('redColor');
    		ele.addClass("err_border");
    	} else {
    		//$("#" + ele.attr("id") + "_err").hide();
        $("#" + ele.attr("id") + "_err").removeClass('redColor');
    		ele.removeClass("err_border");
    	}
    	return;
    }
    var minMaxFlag = false;
    if(ele.attr("min_length")){
    	if(currVal && currVal.length < ele.attr("min_length")) minMaxFlag = true;
    }
    if(ele.attr("max_length")){
    	if(currVal && currVal.length > ele.attr("max_length")) minMaxFlag = true;
    }
    if (minMaxFlag) {
        //$("#" + ele.attr("id") + "_err").show();
        $("#" + ele.attr("id") + "_err").addClass('redColor');
        ele.addClass("err_border");
        return;
    } else {
        //$("#" + ele.attr("id") + "_err").hide();
        $("#" + ele.attr("id") + "_err").removeClass('redColor');
        ele.removeClass("err_border");
    }
    
    var currValArrStr = "";
    for(var i = 0;i<currValArr.length;i++){
    	
    	 /*<%--先判断是否为整型--%>*/
    	if (reg.test(currValArr[i])) {
    		currValArrStr = parseInt(currValArr[i]);
        } else {
        /*<%--不是整型，显示错误，返回--%>*/
            //$("#" + ele.attr("id") + "_err").show();
            $("#" + ele.attr("id") + "_err").addClass('redColor');
            ele.addClass("err_border");
            return;
        }
    	/*<%--再判断取值范围是否合法--%>*/
        var showFlag = false;
        if (ele.attr("min_value")) {
        /*<%--有最小值--%>*/
            var minValue = parseInt(ele.attr("min_value"), 10);
            if (currValArrStr < minValue) {
                showFlag = true;
            }
        }
        if (ele.attr("max_value")) {
        /*<%--有最大值--%>*/
            var maxValue = parseInt(ele.attr("max_value"), 10);
            if (currValArrStr > maxValue) {
                showFlag =true;
            }
        }
        if (showFlag) {
            //$("#" + ele.attr("id") + "_err").show();
            $("#" + ele.attr("id") + "_err").addClass('redColor');
            ele.addClass("err_border");
        } else {
            //$("#" + ele.attr("id") + "_err").hide();
            $("#" + ele.attr("id") + "_err").removeClass('redColor');
            ele.removeClass("err_border");
        }
    	
    	
    }      
}

/*<%--验证是否为数字，是否超出最大值，是否超出最小值--%>*/
function validateMaxAndMinVal_double(e) {
    var ele = $(e["target"]);
    var reg = /^-?\d+\.?\d*$/;
    var currVal = ele.val();
    var must = ele.attr("must");
    var currValLength = currVal.length;
    if (currValLength == 0) {
    	if (must == 1) {
    		//$("#" + ele.attr("id") + "_err").show();
        $("#" + ele.attr("id") + "_err").addClass('redColor');
    		ele.addClass("err_border");
    	} else {
    		//$("#" + ele.attr("id") + "_err").hide();
        $("#" + ele.attr("id") + "_err").removeClass('redColor');
    		ele.removeClass("err_border");
    	}
    	return;
    }
    var reg2 = /\d+\.$/;
    
    /*<%--先判断是否为浮点型--%>*/
    if (reg.test(currVal) && !reg2.test(currVal)) {
        currVal = parseFloat(currVal);
    } else {
    /*<%--不是浮点型，显示错误，返回--%>*/
        //$("#" + ele.attr("id") + "_err").show();
        $("#" + ele.attr("id") + "_err").addClass('redColor');
        ele.addClass("err_border");
        return;
    }
    /*<%--再判断取值范围是否合法--%>*/
    var showFlag = false;
    if (ele.attr("min_value")) {
    /*<%--有最小值--%>*/
        var minValue = parseFloat(ele.attr("min_value"), 10);
        if (currVal < minValue) {
            showFlag = true;
        }
    }
    if (ele.attr("max_value")) {
    /*<%--有最大值--%>*/
        var maxValue = parseFloat(ele.attr("max_value"), 10);
        if (currVal > maxValue) {
            showFlag =true;
        }
    }
    if (showFlag) {
        //$("#" + ele.attr("id") + "_err").show();
        $("#" + ele.attr("id") + "_err").addClass('redColor');
        ele.addClass("err_border");
    } else {
        //$("#" + ele.attr("id") + "_err").hide();
        $("#" + ele.attr("id") + "_err").removeClass('redColor');
        ele.removeClass("err_border");
    }
}
// 根据正则表达式验证并判断大小
function validateByRegexAndRange(e) {
	var ele = $(e["target"]);
    var reg = eval(ele.attr("vali-regex"));
    var currVal = ele.val();
    var must = ele.attr("must");
    var minValue = ele.attr("min_value");
    var maxValue = ele.attr("max_value");
    
    var currValLength = currVal.length;
    if (currValLength == 0) {
    	if (must == 1) {
    		//$("#" + ele.attr("id") + "_err").show();
        $("#" + ele.attr("id") + "_err").addClass('redColor');
    		ele.addClass("err_border");
    	} else {
    		//$("#" + ele.attr("id") + "_err").hide();
        $("#" + ele.attr("id") + "_err").removeClass('redColor');
    		ele.removeClass("err_border");
    	}
    	return;
    }
    
    var showFlag = false;
    if(reg){
	    if (!reg.test(currVal)) {
	    	showFlag = true;
	    }else{
	    	if(currVal.indexOf("..")>-1){
				currVal= currVal.split("..");
				if(currVal[1]-currVal[0]<=0){
					showFlag = true;
				}
				if(minValue-currVal[0]>0){
					showFlag = true;
				}
				if(maxValue-currVal[1]<0){
					showFlag = true;
				}
			}else{
				if(minValue-currVal>0){
					showFlag = true;
				}
				if(maxValue-currVal<0){
					showFlag = true;
				}
			}
	    }
    }
    
    var err_prompt_id = ele.attr("err_prompt_id");
    if (showFlag) {
        //$("#" + ele.attr("id") + "_err").show();
        $("#" + ele.attr("id") + "_err").addClass('redColor');
        ele.addClass("err_border");
        if (err_prompt_id) {
        	$("#" + err_prompt_id).show();
        }
    } else {
        //$("#" + ele.attr("id") + "_err").hide();
        $("#" + ele.attr("id") + "_err").removeClass('redColor');
        ele.removeClass("err_border");
        if (err_prompt_id) {
        	$("#" + err_prompt_id).hide();
        }
    }
}

// 根据正则表达式验证
function validateByRegex(e) {
	var ele = $(e["target"]);
    var reg = eval(ele.attr("vali-regex"));
    var currVal = ele.val();
    var must = ele.attr("must");
    
    var currValLength = currVal.length;
    if (currValLength == 0) {
    	if (must == 1) {
    		//$("#" + ele.attr("id") + "_err").show();
        $("#" + ele.attr("id") + "_err").addClass('redColor');
    		ele.addClass("err_border");
    	} else {
    		//$("#" + ele.attr("id") + "_err").hide();
        $("#" + ele.attr("id") + "_err").removeClass('redColor');
    		ele.removeClass("err_border");
    	}
    	return;
    }
    
    
    var showFlag = false;
    if(reg){
	    if (!reg.test(currVal)) {
	    	showFlag = true;
	    }else{
	    	if(currVal.indexOf("..")>-1){
				currVal= currVal.split("..");
				if(currVal[1]-currVal[0]<=0){
					showFlag = true;
				}
			}
	    
	    }
    }
    
    var err_prompt_id = ele.attr("err_prompt_id");
    if (showFlag) {
        //$("#" + ele.attr("id") + "_err").show();
        $("#" + ele.attr("id") + "_err").addClass('redColor');
        ele.addClass("err_border");
        if (err_prompt_id) {
        	$("#" + err_prompt_id).show();
        }
    } else {
        //$("#" + ele.attr("id") + "_err").hide();
        $("#" + ele.attr("id") + "_err").removeClass('redColor');
        ele.removeClass("err_border");
        if (err_prompt_id) {
        	$("#" + err_prompt_id).hide();
        }
    }
}

/**
 * 验证是否为合法的MAC地址
 */
function validateMACAddress(e) {
	var reg = /^([0-9a-fA-F]{2})(([:-][0-9a-fA-F]{2}){5})$/;
	var ele = $(e["target"]);
	
	if (ele.val().length == 0) {
		var must = ele.attr("must");
		if (must == "1") {
			//$("#" + ele.attr("id") + "_err").show();
        $("#" + ele.attr("id") + "_err").addClass('redColor');
	      ele.addClass("err_border");
		} else {
			//$("#" + ele.attr("id") + "_err").hide();
        $("#" + ele.attr("id") + "_err").removeClass('redColor');
	      ele.removeClass("err_border");
		}
		return;
	}
	
	if (reg.test(ele.val())) {// 格式正确
		//$("#" + ele.attr("id") + "_err").hide();
      $("#" + ele.attr("id") + "_err").removeClass('redColor');
      ele.removeClass("err_border");
	} else {
		//$("#" + ele.attr("id") + "_err").show();
      $("#" + ele.attr("id") + "_err").addClass('redColor');
      ele.addClass("err_border");
	}
}

/* 验证邮箱地址 */
function validateEmail(val) {
    if (val == null || val == "") {
        return true;
    }
    //var reg = /^[\S]+@[\S]+$/;
    var reg = /^([a-zA-Z0-9_\.\-])+@([a-zA-Z0-9_-])+(\.[a-zA-Z0-9_-]+)+$/;
    //var reg = /^(([a-z0-9_\.-]+)@([\da-z\.-]+)\.([a-z\.]{2,6}\;))*([a-z0-9_\.-]+)@([\da-z\.-]+)\.([a-z\.]{2,6})$/;
    return reg.test(val)
}

/**
 * 验证是否为合法的IP地址
 */
function validateIPAddress(e) {
	var reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
	var ele = $(e["target"]),
		visible = isVisible(ele[0]);
	
	if (ele.val().length == 0) {
		var must = ele.attr("must");
		if (must == "1" && visible) {
			//$("#" + ele.attr("id") + "_err").show();
			$("#" + ele.attr("id") + "_err").addClass('redColor');
	        ele.addClass("err_border");
		} else {
			//$("#" + ele.attr("id") + "_err").hide();
			$("#" + ele.attr("id") + "_err").removeClass('redColor');
	        ele.removeClass("err_border");
		}
		return;
	}
	
	if (isValidIP(ele.val()) || !visible) {// 格式正确
		//$("#" + ele.attr("id") + "_err").hide();
		$("#" + ele.attr("id") + "_err").removeClass('redColor');
		ele.removeClass("err_border");
	} else {
		//$("#" + ele.attr("id") + "_err").show();
		$("#" + ele.attr("id") + "_err").addClass('redColor');
		ele.addClass("err_border");
	}
}
function trimValue(){
	$(this).val($(this).val().trim());
}
//Array对象的indexOf()方法来取得这个元素在当前数组中的索引值，若索引值不等于-1，数组中就存在这个元素
//但是IE9以前的版本都不支持此方法，这里加扩展来兼容  add by xuhm
Array.prototype.indexOf = function(el){
	for (var i=0,n=this.length; i<n; i++){
		if (this[i] === el){
			return i;
		}
	}
	return -1;
}

/* easyui 增加扩展    start */

//树获取level ar lv =  $().tree("getLevel",node.target);
$.extend($.fn.tree.methods, {
	    getLevel:function(jq,target){
	        var l = $(target).parentsUntil("ul.tree","ul");
	        return l.length+1;
	    }
});
//
/* easyui 增加扩展    end */

/*动态调用方法、并传递参数    
	doCallback(eval("callback2"),['a','b']); 
*/
function doCallback(fn,args)    
{    
    fn.apply(this, args);  
}    
function agAlert(msg, flg, title) {
	title = title || TiShi;
	flg = flg || '';
	$.messager.alert(title,msg, flg);
}

// json 数值转换为html request参数串
function json2htmlParam(data) {
	var htmlParam = "";
	if (data) {
		for ( var key in data) {
			htmlParam += "&" + key + "=" + data[key];
		}
	}
	return htmlParam;
}

function easyUiWinClose(open_win_id) {	
	try {
		$('.panel-tool-close').each(function() {
			//alert( $(this).parent().parent().parent().html() );
			if ( $(this).parent().parent().parent().find('#'+open_win_id).length > 0 ) {
				$(this).click();
			}					
		});
	} catch(e) { }
}

//判断输入中是否含有非法字符
function illeagal(str){
	var pat=new RegExp("[@<>%&',;=\*\$\^]","g");   
      if(pat.test(str)==true){   
          //alert("输入中含有非法字符");   
          return true;   
      }  
      return false;
}

/*查看treeWidget*/
function aiTreeWidget(jsonParam){

	var urlParm = json2htmlParam(jsonParam);	
	var winid= jsonParam.open_win_id || 'treeWidgetWin';
	jsonParam.open_win_id = winid;
	var url = site_url+"/mts/global/treeWidget.action?a=1"+urlParm;//业务逻辑页面  url
	//var winid= jsonParam.open_win_id || 'treeWidgetWin';
	var wdt_frm_url = site_url+"/system/sysGlb/wdtFrm.action?a=1"+urlParm;// div嵌套frm页面 open_win_id 必须
	var load_type = jsonParam.load_type || 'load_flash'; // load_cache  ||  load_flash
	easyUiWinLoadType(load_type,winid,url, wdt_frm_url); 
}

function sysTreeWidgetUtil(jsonParam){

	var urlParm = json2htmlParam(jsonParam);	
	var winid= jsonParam.open_win_id || 'sysTreeWidgetWin';
	jsonParam.open_win_id = winid;
	var url = site_url+"/system/sysGlb/treeWidget.action?a=1"+urlParm;		
	var wdt_frm_url = site_url+"/system/sysGlb/wdtFrm.action?a=1"+urlParm;// div嵌套frm页面 open_win_id 必须
	var load_type = jsonParam.load_type || 'load_flash'; // load_cache  ||  load_flash
	easyUiWinLoadType(load_type,winid,url, wdt_frm_url); 		
}


function sysGridWidgetUtil(jsonParam){

	var urlParm = json2htmlParam(jsonParam);		
	var winid= jsonParam.open_win_id || 'adm_info_win';
	jsonParam.open_win_id = winid;
	var url = site_url+"/system/sysGlb/sysDgridWdt.action?a=1"+urlParm;//业务逻辑页面  url	
	var wdt_frm_url = site_url+"/system/sysGlb/wdtFrm.action?a=1"+urlParm;;// div嵌套frm页面 open_win_id 必须
	var load_type = jsonParam.load_type || 'load_flash'; // load_cache  ||  load_flash
	easyUiWinLoadType(load_type,winid,url, wdt_frm_url); 
			
}

function easyUiWinLoadType(load_type, winid, url, wdt_frm_url) {
	
	load_type = load_type || 'load_flash'; // load_cache  ||  load_flash
	if ( load_type == 'load_flash' ) {
		//每次都重新刷新frame
		$('#'+winid).attr("frm_url", url);//设置iframe请求url到div属性"frm_url"
		$('#'+winid).window('open');
		$('#'+winid).window('refresh', wdt_frm_url);
	} else {
		if ( $('#'+winid).attr("frm_url") == url ) {//刷新过frame就不用重新刷,
			
			$('#'+winid).window('open');
		} else { //变化了url，要重新加载
			$('#'+winid).attr("frm_url", url);//设置iframe请求url到div属性"frm_url"
			$('#'+winid).window('open');
			$('#'+winid).window('refresh', wdt_frm_url);
		}		
	}	
}


function fnChangeCity(city_id, val){
	$("#"+city_id).combobox('reload',site_url+'/system/sysGlb/findSysRegn.action?is_all=1&type=2&param='+val);
}
function fnChangeDist(dist_id,val){
	$("#"+dist_id).combobox('reload',site_url+'/system/sysGlb/findSysRegn.action?is_all=1&type=3&param='+val);
}

/*<%-- 在右下角提示显示、或显示在上方面板中 --%>*/
function showAtPromptOrPanel(element, netType) {
  if([undefined, null, ''].includes(netType)) {
    var nav = $('.navigator-ctn .el-tabs__item.is-active'),
      clsId = nav.attr('id').replace('tab', 'pane'),
      ctn = $('#' + clsId);

    if ($("#showParamValues", ctn).length > 0) {
        $("#showParamValues", ctn).append(element);
        //<%--将滚动条滚到最下方--%>
        $("#showParamValues", ctn).animate({scrollTop: $("#showParamValues", ctn)[0].scrollHeight + 'px'}, 500);
        $('.param-values-oper', ctn).removeClass('empty');
    } 
    
    if ($("#showParamValues_gnb", ctn).length > 0) {
      $("#showParamValues_gnb", ctn).append(element);
      //<%--将滚动条滚到最下方--%>
      $("#showParamValues_gnb", ctn).animate({scrollTop: $("#showParamValues_gnb", ctn)[0].scrollHeight + 'px'}, 500);
      $('.param-values-oper', ctn).removeClass('empty');
    }
  }else {
    if(netType == 'gnb') {
      $("#showParamValues_gnb").append(element);
      //<%--将滚动条滚到最下方--%>
      $("#showParamValues_gnb").animate({scrollTop: $("#showParamValues_gnb")[0].scrollHeight + 'px'}, 500);
      $('#MML_Config_gnb .param-values-oper').removeClass('empty');
    }else {
      $("#showParamValues").append(element);
      //<%--将滚动条滚到最下方--%>
      $("#showParamValues").animate({scrollTop: $("#showParamValues")[0].scrollHeight + 'px'}, 500);
      $('#enbmmlResult .param-values-oper').removeClass('empty');
    }
  }
}

function showErrorMML(data, operName, objectParam){
	if(data["errorMML"] && data["errorMML"].length > 0){
		var divEle = $("<div class='getSetParamValResult'></div>");
	    divEle.append($("<span class='resultTitle'>" + operName + ":</span><br/>"));
	    
	    /*
	    for(var i=0; i < data["errorMML"].length; i++){
	    	var arr = data["errorMML"][i].split(" ");
	    	if(arr[0] == "ACT"){
	    		divEle.append($("<span>" + data["errorMML"][i] + ", please do it later." + "</span><br/>"));
	    	}else{
	    		divEle.append($("<span>" + data["errorMML"][i] + "</span><br/>"));
	    	}
	    }*/
	    for(var i=0; i < data["errorMML"].length; i++){
	    	divEle.append($("<span>" + data["errorMML"][i] + "</span><br/>"));
	    }
	    
	   /* divEle.append($("<span>" + CuoWuMa + MaoHao + data[i]["faultCode"] + "</span><br/>"));
		var lineDiv = $("<span class='lineDiv' style='font-weight: normal;font-size: 15px;'>" + CuoWuXinXi + MaoHao + WuPeiZhiXinXi+"</span>");
		divEle.append(lineDiv);*/
		
		showAtPromptOrPanel(divEle);//展示信息
		
		if(!isShowRightClickMenu){
	    	 // 配置结果右键事件 
	    	isShowRightClickMenu = true;
	    }
	    
	}
}
/*<%-- 参数查询结果 --%>*/
function showGetParamVal(data, operName, objectParam, netType){
	/*<%-- 将返回来的数据进行组装，并显示；如果处于设置页面，则直接在上方的显示板中显示；如果没有，则在右下角弹出提示来显示结果 --%>*/
	var divEle = $("<div class='getSetParamValResult'></div>");
    divEle.append($("<span class='resultTitle'>" + operName + " RESULT:</span>"));

	for (var i = 0; i < data.length; i++) {
		divEle.append($("<br/><span>SmallCell: " + data[i]["cell"] + "</span><br/>"));
		if (objectParam) {
			//判断是否是邻区邻频，如果是邻区邻频，查询出错，则认为没有配置
			/*var qpArr = ["LST EUTRANNFREQ","LST TDSNFREQ","LST GSMNFREQ","LST EUTRANNCELL","LST TDSNCELL","LST GSMNCELL"];
			if (data[i]["faultCode"] && data[i]["faultCode"] == "9005") {
				var flag = false;
				 $.each(qpArr,function(index,ele){
                     if(ele == operName){
                    	 flag = true;
                     }     					 
				 });
				if(flag){
					var lineDiv = $("<span class='lineDiv' style='font-weight: normal;font-size: 15px;'>"+WuPeiZhiXinXi+"</span>");
					divEle.append(lineDiv);
					showAtPromptOrPanel(divEle);//展示信息
					
					if(!isShowRightClickMenu){
				    	  配置结果右键事件 
				    	isShowRightClickMenu = true;
				    }
					return;
				}
			}*/
			
			
			if(data[i]["faultCode"] == "9005"){
				divEle.append($("<span>" + CuoWuMa + MaoHao + data[i]["faultCode"] + "</span><br/>"));
				var lineDiv = $("<span class='lineDiv' style='font-weight: normal;font-size: 15px;'>" + CuoWuXinXi + MaoHao + WuPeiZhiXinXi+"</span>");
				divEle.append(lineDiv);
				continue;
			}
			
			jQuery.each(data[i]["paramVals"], function(index, element){
				var lineDiv = $("<div class='lineDiv' style='display: flex;'></div>");
				lineDiv.append($("<div class='cell instanceNum' style='padding: 2px 5px;border: 1px solid #BBBBBB;'><span class='key'>"+XuLieHao+"</span><br/><span class='value'>" + index + "</span></div>"));

				jQuery.each(element, function(name, value){
					lineDiv.append($("<div class='cell' style='padding: 2px 5px;border: 1px solid #BBBBBB;'><span class='key'>" + name + "</span><br/><span class='value'>" + value + "</span></div>"));
				});
				divEle.append(lineDiv);
			});
		} else {
			if (data[i]["paramVals"]) {
				// 显示参数值
				divEle.append(createEleByParamValsList(data[i]["paramVals"]));
			}
		}

		/*<%-- 显示错误信息 --%>*/
		if (data[i]["faultCode"]) {
			divEle.append($("<span>" + CuoWuMa + MaoHao + data[i]["faultCode"] + "</span><br/>"));
			if(data[i]["faultCode"] == "1001"){
				divEle.append($("<span>" + CuoWuXinXi + MaoHao + XiaFaMingLingChaoShi + "</span><br/>"));
			}else{
				divEle.append($("<span>" + CuoWuXinXi + MaoHao + data[i]["faultString"] + "</span><br/>"));
			}
			
		}
		//divEle.append($("<hr>"));
	}

    showAtPromptOrPanel(divEle, netType);
    
    if(!isShowRightClickMenu){
    	 /* 配置结果右键事件 */
    	isShowRightClickMenu = true;
    }
}
/*<%-- 生成参数名、参数值的JQUERY元素 --%>*/
function createEleByParamValsList(paramValsList) {
	
	var nameValDiv = $("<div style='display: flex;'></div>");
	if (paramValsList.length > 0) {
		for (var i = 0; i < paramValsList.length; i++) {
			//if (i == 0) {
				nameValDiv.append($("<div class='cell line-left' style='padding: 2px 5px;'><span class='key'>" + paramValsList[i]["name"] + "</span><br/><span class='value'>" + paramValsList[i]["val"] + "</span></div>"));
			//} else {
			//	nameValDiv.append($("<div class='cell' style='padding: 2px 5px;'><span class='key'>" + paramValsList[i]["name"] + "</span><br/><span class='value'>" + paramValsList[i]["val"] + "</span></div>"));
			//}
		}
	} else {
		//nameValDiv.append($("<span>" + HuoQuBuDaoCanShuZhi + "</span><br/>"));
	}

	return nameValDiv;
}
/*<%-- 设置参数的结果 --%>*/
function showSetParamVal(data, operName, objectParam, index, netType) {
    var divEle = $("<div class='getSetParamValResult'></div>");
    divEle.append($("<span class='resultTitle'>" + operName + " RESULT:</span><br/>"));
    for (var i = 0; i < data.length; i++) {
        divEle.append($("<span>SmallCell: " + data[i]["cell"] + data[i]["runResult"] + "</span><br/>"));
        if (data[i]["paramVals"]) {
        	if (objectParam) {
        		var lineDiv = $("<div class='lineDiv' style='display: flex;'></div>");
        		if(index != null && !data[i]["faultCode"]){//如果index不为null且没有报错，则显示index
        			lineDiv.append($("<div class='cell instanceNum'><span class='key'>"+XuLieHao+"</span><br/><span class='value'>" + index + "</span></div>"));
        		}
				for (var j = 0; j < data[i]["paramVals"].length; j++) {
					if(j == 0){
						//如果是第一个参数，其左边框也添加上border
						lineDiv.append($("<div class='cell cell2'><span class='key' style='padding:8px 5px;'>" + data[i]["paramVals"][j]["name"] + "</span><br/><span class='value'>" + data[i]["paramVals"][j]["val"] + "</span></div>"));
					}else{
						//除第一个参数外，后面的参数的左边框就是上一个参数的有边框
						lineDiv.append($("<div class='cell cell2'><span class='key'>" + data[i]["paramVals"][j]["name"] + "</span><br/><span class='value'>" + data[i]["paramVals"][j]["val"] + "</span></div>"));
					}
				}
				divEle.append(lineDiv);
        	} else {
        		divEle.append(createEleByParamValsList(data[i]["paramVals"]));        		
        	}
        }
        if (data[i]["faultCode"]) {
        	divEle.append($("<span>" + CuoWuMa + MaoHao + data[i]["faultCode"] + "</span><br/>"));
            if(data[i]["faultCode"]=="1001"){
            	divEle.append($("<span>" + CuoWuXinXi + MaoHao + XiaFaMingLingChaoShi + "</span><br/>"));
            }else{
            	divEle.append($("<span>" + CuoWuXinXi + MaoHao + data[i]["faultString"] + "</span><br/>"));
            }
        }
        //divEle.append($("<hr>"));
    }
    showAtPromptOrPanel(divEle, netType);
    
    if(!isShowRightClickMenu){
   	 /* 配置结果右键事件 */
   	 isShowRightClickMenu = true;
    }
}

function showRebootInfo(data,operName, netType){
	var divEle = $("<div class='getSetParamValResult'></div>");
    divEle.append($("<span class='resultTitle'>" + operName + " RESULT:</span><br/>"));
    for (var i = 0; i < data.length; i++) {
    	divEle.append($("<span>SmallCell: " + data[i]["smallCellCode"] + "</span><br/>"));
        if (data[i]["faultCode"]) {
        	divEle.append($("<span>" + CuoWuMa + MaoHao + data[i]["faultCode"] + "</span><br/>"));
            if(data[i]["faultCode"] == "1001"){
            	divEle.append($("<span>" + CuoWuXinXi + MaoHao + XiaFaMingLingChaoShi + "</span><br/>"));
            }else{
            	divEle.append($("<span>" + CuoWuXinXi + MaoHao + data[i]["faultString"] + "</span><br/>"));
            }
            
        }else{
        	divEle.append($("<span style='display:inline;margin-top:10px;'>" +ChongQiWanChengBingLianJieDaoOMC+ "</span><br/>"));
        }
    }
    showAtPromptOrPanel(divEle, netType);

	if (!isShowRightClickMenu) {
	  /* 配置结果右键事件 */
	  isShowRightClickMenu = true;
	}
}
function showResetInfo(data,operName, netType){
	var divEle = $("<div class='getSetParamValResult'></div>");
    divEle.append($("<span class='resultTitle'>" + operName + " RESULT:</span><br/>"));
    for (var i = 0; i < data.length; i++) {
    	divEle.append($("<span>SmallCell: " + data[i]["smallCellCode"] + "</span><br/>"));
        if (data[i]["faultCode"]) {
        	divEle.append($("<span>" + CuoWuMa + MaoHao + data[i]["faultCode"] + "</span><br/>"));
            if(data[i]["faultCode"] == "1001"){
            	divEle.append($("<span>" + CuoWuXinXi + MaoHao + XiaFaMingLingChaoShi + "</span><br/>"));
            }else{
            	divEle.append($("<span>" + CuoWuXinXi + MaoHao + data[i]["faultString"] + "</span><br/>"));
            }
            
        }else{
        	divEle.append($("<span style='display:inline;margin-top:10px;'>" +HuiFuChuChangSheZhiWanChengBingLianJieDaoOMC+ "</span><br/>"));
        }
    }
    showAtPromptOrPanel(divEle, netType);
    
    if(!isShowRightClickMenu){
      	 /* 配置结果右键事件 */
      isShowRightClickMenu = true;
     }
}

/**
 * 显示命令行执行结果
 * @param data
 */
function showCommandResult(data) {
	if ($(".cmd_result").length > 0) {
		var line = $("<span>" + data + "<span><br/>");
		$(".cmd_result:last").append(line);
		/*<%--将滚动条滚到最下方--%>*/
        $("#showParamValues").animate({scrollTop: $("#showParamValues")[0].scrollHeight + 'px'}, 500);
	}
	
	if(!isShowRightClickMenu){
	   	 /* 配置结果右键事件 */
	   isShowRightClickMenu = true;
	}
}

/*<%-- AddObject完成的通知 --%>*/
function showAddObjectCompleteTip(data, operName, netType) {
	var divEle = $("<div class='getSetParamValResult'></div>");
    divEle.append($("<span class='resultTitle'>" + operName + " RESULT:</span><br/>"));
    for (var i = 0; i < data.length; i++) {
    	divEle.append($("<span class='smallcell'>SmallCell: " + data[i]["cell"] + "</span><br/>"));
        if (data[i]["instanceNumber"] && !data[i]["faultCode"]) {
        	divEle.append($("<span>" + XuLieHao + MaoHao + data[i]["instanceNumber"] + "</span><br/>"));
        	if (data[i]["status"] == "0") {
        		divEle.append($("<span>" + ZhuangTaiJianLi + "</span><br/>"));
        	} else {
				divEle.append($("<span>" + ZhuangTaiTiJiaoDanMeiYingYong+ "</span><br/>"));
        	}
        }
        if (data[i]["faultCode"]) {
        	divEle.append($("<span>" + CuoWuMa + MaoHao + data[i]["faultCode"] + "</span><br/>"));
            if(data[i]["faultCode"] == "1001"){
            	divEle.append($("<span>" + CuoWuXinXi + MaoHao + XiaFaMingLingChaoShi + "</span><br/>"));
            }else{
            	divEle.append($("<span>" + CuoWuXinXi + MaoHao + data[i]["faultString"] + "</span><br/>"));
            }
        }
    }
    showAtPromptOrPanel(divEle, netType);
    
    if(!isShowRightClickMenu){
      	 /* 配置结果右键事件 */
      isShowRightClickMenu = true;
    }
}
/*<%-- DeleteObject完成的通知 --%>*/
function showDeleteObjectCompleteTip(data, operName, netType) {
	var divEle = $("<div class='getSetParamValResult'></div>");
    divEle.append($("<span class='resultTitle'>" + operName + " RESULT:</span><br/>"));
    for (var i = 0; i < data.length; i++) {
    	divEle.append($("<span class='smallcell'>SmallCell: " + data[i]["cell"] + "</span><br/>"));
    	divEle.append($("<span>" + XuLieHao + MaoHao + data[i]["indexNumber"] + "</span><br/>"));
        if (data[i]["status"]) {
        	if (data[i]["status"] == "0") {
        		divEle.append($("<span>" + ZhuangTaiShanChu + "</span><br/>"));
        	} else {
        		divEle.append($("<span>" +ZhuangTaiTiJiaoDanMeiYingYong+ "</span><br/>"));
        	}
        }
        if (data[i]["faultCode"]) {
        	divEle.append($("<span>" + CuoWuMa + MaoHao + data[i]["faultCode"] + "</span><br/>"));
            if(data[i]["faultCode"] == "1001"){
            	divEle.append($("<span>" + CuoWuXinXi + MaoHao + XiaFaMingLingChaoShi + "</span><br/>"));
            }else{
            	divEle.append($("<span>" + CuoWuXinXi + MaoHao + data[i]["faultString"] + "</span><br/>"));
            }
           
        }
    }
    showAtPromptOrPanel(divEle, netType);
    
    if(!isShowRightClickMenu){
      	 /* 配置结果右键事件 */
      isShowRightClickMenu = true;
    }
}
/*<%-- 踢除用户 --%>*/
function kickOutUser(force_user_code) {
    if (force_user_code.toLowerCase() == user_code.toLowerCase()) {
        $.messager.alert(TiShi, BeiQiangZhiTuiChu);
        setTimeout("kickOut()", 3000);
    }
}

function logoutUserByGroupDestroyed(force_user_code) {
	if (force_user_code == user_code) {
        $.messager.alert(TiShi, ZuBeiShanChuYongHuTuiChu);
        setTimeout("kickOut()", 3000);
    }
}

function kickOut(){
	logout(webRootPath);
}

function toggleEffect(ele) {
    if ($(ele).hasClass("selectEffect")) {
        return;
    }
    $(ele).siblings(".selectEffect").removeClass("selectEffect");
    $(ele).addClass("selectEffect");
}

/*<%--格式化时间--%>*/
function dateformatter(date) {
    var y = date.getFullYear();
    var m = date.getMonth() + 1;
    var d = date.getDate();
    var h = date.getHours();
    var min = date.getMinutes();
    var s = date.getSeconds();
    return y + '-' + (m < 10 ? ('0' + m) : m) + '-' + (d < 10 ? ('0' + d) : d) + " " + (h < 10 ? ('0' + h) : h) + ":" + (min < 10 ? ('0' + min) : min) + ":" + (s < 10 ? ('0' + s) : s);
}

function dateboxFormatter(date) {
	var y = date.getFullYear();
    var m = date.getMonth() + 1;
    var d = date.getDate();
    return y + '-' + (m < 10 ? ('0' + m) : m) + '-' + (d < 10 ? ('0' + d) : d);
}

function dateParser(s) {
	if (!s) {
		return new Date(gloableTime.replace(/-/g,'/'));
	}
	s = s.replace(/-/g,'/');
	var reg = /(\d+)\/(\d+)\/(\d+) (\d+):(\d+):(\d+)/;
	var arr = reg.exec(s);
	return new Date(arr[1], arr[2] - 1, arr[3], arr[4], arr[5], arr[6]);
}

function dateboxParser(s) {
	if (!s) {
		return new Date(gloableTime.replace(/-/g,'/'));
	}
	s = s.replace(/-/g,'/');
	var reg = /(\d+)\/(\d+)\/(\d+)/; 
	var arr = reg.exec(s);
	return new Date(arr[1], arr[2] - 1, arr[3]);
}

$.fn.datetimebox.defaults.formatter = dateformatter;
$.fn.datetimebox.defaults.parser = dateParser;
$.fn.datebox.defaults.formatter = dateboxFormatter;
$.fn.datebox.defaults.parser = dateboxParser;
$.fn.window.defaults.resizable = false;
$.fn.window.defaults.draggable = false;

// 禁用树中复选框
function diableTreeCheckbox(tree_id) {
	$("#" + tree_id).tree("options")["onBeforeCheck"] = function() {
		if($(this).tree("options")["checkboxDisable"]) {
    		return false;
    	}
	}
	$("#" + tree_id).tree("options")["checkboxDisable"] = true;
	$("#" + tree_id).addClass("checkboxDisableTree");
}

// 禁用后，重新启用树中复选框
function enableTreeCheckbox(tree_id) {
	if ($("#" + tree_id).hasClass("checkboxDisableTree")) {
		$("#" + tree_id).removeClass("checkboxDisableTree");
		$("#" + tree_id).tree("options")["checkboxDisable"] = false;
	}
}

//修改easyuidatagrid加载提示
$.fn.datagrid.defaults.loadMsg = 'Loading ...';

// 上传文件进度
var intervalGetProgress;
function getUploadProgress() {
    $.post(webRootPath + '/fileUpload/getProgress.action', {}, function (data) {
        var progress = parseInt(data["progress"]);
        // 更新进度条
        $('#progressUploadFile').progressbar('setValue', progress);
        if (progress == 100) {
            // 清除定时器
            window.clearInterval(intervalGetProgress);
        }
    }, "json");
}
// 搜索框样式
function inputOnfocusStyle(ele){
	$(ele).css('border-bottom','1px solid #209FFF');
}
function inputOnBlourStyle(ele){
	$(ele).css('border-bottom','1px solid #D0D9DE');
}

function changeColor_select(){
	var opts = $(this).find("option");
	//需要对OLD的命名规则为mmepool_OLD_mibdn,此时发过来的id为mmepool_mibdn，需要转换
	var oldValue = $(this).attr("oldValue");
	var nowValue = $(this).val();
	for(var i=0;i<opts.length;i++){
		if(opts[i].value == nowValue){
			$(opts[i]).attr("selected",true);
			if(oldValue!=nowValue){
				$(this).css({"color":"blue"});
				opts.css({"color":"black"});
			}else{
				$(this).css({"color":"black"});	
			}
			break;
		}
	}
}
function changeColor_input(){
    var currVal = $(this).val();
    var oldValue = $(this).attr("oldValue");
    if(currVal!=oldValue){
    	$(this).css({"color":"blue"});
    }else{
    	$(this).css({"color":"black"});
    }
}

function isICICConfigEnable(e){
	var ele = $(e["target"]);
	if(ele.val() == '1'){
		$("#icicDetailConfig").show();
	}else{
		$("#icicDetailConfig").hide();
	}
}

function validateTimePeriod(startTimeStr, endTimeStr){
	if (isNotNull(startTimeStr) && isNotNull(endTimeStr)) {
        var startDate = dateParser(startTimeStr);
        var endDate = dateParser(endTimeStr);
        if((endDate -startDate)/(1000*60*60*24)>31){
        	return "false";
        }
    }
}

/*EPC全局遮罩*/
function savingCover(){
	  $("#epcWinLoadingPro").show();
	  $("#loadingCover").show();
}
function cancelSavingCover(){
	  $("#epcWinLoadingPro").hide();
	  $("#loadingCover").hide();
}

//获取前几天的日期
function getYesterDay(n){
	 var time=24*60*60*1000;
	 var newdate = null;
	 var now = new Date(gloableTime.substring(0,10)+' 00:00:00');
	 var nowYear = now.getFullYear();
	 var nowMonth = now.getMonth();
	 var nowDate = now.getDate();
	 newdate = new Date(nowYear,nowMonth,nowDate,12,0,0);
	 var newtimems=newdate.getTime()-n*time;
	 var yesd = new Date(newtimems);
	 var yesYear = yesd.getFullYear();
	 var yesMonth = yesd.getMonth();
	 var yesDate = yesd.getDate();
	 yesMonth = doHandleMonth(yesMonth + 1);
	 yesDate = doHandleMonth(yesDate);
	 return yesYear+"-"+yesMonth+"-"+yesDate;
}
function doHandleMonth(month){
	 if(month.toString().length == 1){
	  month = "0" + month;
	 }
	 return month;
}
//日期偏差  - 分钟
function addTimes(date,times){
	var d = new Date(date);
	d = d.valueOf();
	d = d + times*60*1000;
	a = new Date(d);
	return a;
}
// 日期格式化为字符串
function formatDate(date){
	var h = date.getHours(),
		m = date.getMinutes(),
		s = date.getSeconds(),
		mon = (date.getMonth()+1)>9?(date.getMonth()+1):'0'+(date.getMonth()+1),
	    d = date.getDate()>9?date.getDate():'0'+date.getDate();
	var dateStr = date.getFullYear()+'-'+mon+'-'+d+' '
				  + (h>9?h:'0'+h)+':'+(m>9?m:'0'+m)+':'+(s>9?s:'0'+s);
	return dateStr;
}
// 日期偏移  - 天
/*function addDate(date,num){
	var d = new Date(date);
	d = d.valueOf();
	d = d + num*24*60*60*1000;
	a = new Date(d);
	return a;
}*/
// 日期偏移  - 天
function addDate(date, days){
  let newDate = new Date(date);
  newDate.setDate(newDate.getDate() + days); 
  return newDate;
}
// 月份偏移  - 月
function addMonth(date,num){
  var dateStr = date.length > 7 ? date : date + '-01';
	var d = new Date(dateStr);
	d.setMonth(d.getMonth()+num);
	y = d.getFullYear();
	m = d.getMonth()+1;
	m = m < 10 ? '0' + m : m;
	d = y + "-" + m;
	return d;
}
// 年偏移  - 年
function addYear(date,num){
	var d = new Date(date);
	d.setFullYear(d.getFullYear()+num);
	return d.getFullYear();
}
//获取某天所在的周的周一和周日
function getWeekTime(time){
	var now = new Date(time);
	var nowTime = now.getTime();
	var day = now.getDay()==0 ? 7:now.getDay();
	var oneDayTime = 24*60*60*1000;
	//周一
	var mondayTime = formatDate(new Date(nowTime - (day-1)*oneDayTime)).substring(0,10);
	//周日
	var sundayTime = formatDate(new Date(nowTime + (7-day)*oneDayTime)).substring(0,10);
	var weekTime = {
			'mondayTime':mondayTime,
			'sundayTime':sundayTime
	}
	return weekTime;
}
// 时间差
function differ(pre,after){
	var preDate = new Date(pre), afterDate = new Date(after);
	var differTimes = preDate.valueOf() - afterDate.valueOf();
	return Math.round(differTimes/(24*60*60*1000));
}
/**
   * 自动计算 全不选状态，并设置不选或中间状态
   * @param list: 后台获取的定制信息列表
   * @param cusList: 自定义生成的定制信息列表
   * @param listRef: 后台获取的已定制信息列表属性名
   * @param selRef: 自定义已选中的定制信息列表属性名
   * @param unSelRef: 自定义未选中的定制信息列表属性名
   * @param groupId: 当前选中组的 id
   * @param total: rows的总数
   * @param groupDom: 当前选中组的dom对象
   * @eg: transformData({list:DeviceGroupList,
                        cusList:customGroupData,
                        listRef:'relEnbList',
                        selRef：‘new_seleted_enb’，
                        unSelRef:'new_unseleted_enb',
                        groupId: 30,
                        totalRowsNum: 35,
                        groupDom: $('[value=30]')
                      });
   **/
  function transformData({list,listRef,cusList,selRef,unSelRef,groupId,total,groupDom}){
    groupId +='';
    var data = [], cusData = {unsel:[],sel:[]};

    list.map(function(item){
      if(item.group_id==groupId) data = item[listRef];
    });

    cusList.map(function(item){
      if(item.group_id==groupId) cusData = {sel: item[selRef], unsel: item[unSelRef], isChecked: item.is_checked};
    });
    var groupDom = $(groupDom), isAllUnSel = false;
    if(cusData.unsel.length==0 && cusData.sel.length==0 && data.length==0){
      isAllUnSel = true;
    }
    if(cusData.unsel.length>0){
      if(cusData.sel.length == 0 && isSubArray(data,cusData.unsel)) {
        if(groupDom.attr('change_type') != 'un_to_checked') isAllUnSel = true;
        else if(cusData.unsel.length==total){
          isAllUnSel = true;
        }
      }else{
        if(groupDom.attr('change_type') == 'checked_to_un' && cusData.sel.length == 0 ) isAllUnSel = true;
      }
    }

    if(isAllUnSel) {
      groupDom.prop('checked',false).removeProp('indeterminate');
      return;
    }
    groupDom.prop('indeterminate',true);

    function isSubArray(res,target){
      var isSub = true;
      res.map(function(item){
        if(target.indexOf(item)<0) isSub = false;
      });
      return isSub;
    }
  }
  //点击当前div其他div收起
  function judmentOtherChange(divId,fn){
	if($("#cpeSetting").length>0 ){
		var settingInputText = $("#cpeSetting input");
		var settingInputCheckbox = $("#cpeSetting input[type=checkbox]");
		var settingselect = $("#cpeSetting select");
		var inputTextIsChange = false;
		var inputCheckboxIsChange = false;
		var selectIsChange = false;
		if( $("#cpeSetting").position().left<900){
			$.each(settingInputText,function(index,ele){
				if( $(ele).attr('oldvalue') != $(ele).val()){
					inputTextIsChange = true;
					return false;
				}
			})
			$.each(settingInputCheckbox,function(index,ele){
				if( $(ele).attr('oldValue') != $(ele).attr('value')){
					inputCheckboxIsChange = true;
					return false;
				}
			})
			 $.each(settingselect,function(index,ele){
				if( $(ele).attr('oldvalue') != $(ele).val()){
					selectIsChange = true;
					return false;
				}
			})
		}
		
		if(inputTextIsChange || selectIsChange  || inputCheckboxIsChange){
			$.messager.confirm(QueRen, QueRenBaoCunBianGeng, function (r) {
		        if (r) {
		        	$("#cpeSetting .linkbutton_trend").click();
		        }else{
		        	$.each(settingInputText,function(index,ele){
		        		$(ele).val($(ele).attr('oldvalue'))  ;
		        	})
		        	$.each(settingInputCheckbox,function(index,ele){
		        		$(ele).attr('value',$(ele).attr('oldvalue'))  ;
		        	})
		        	$.each(settingselect,function(index,ele){
		        		$(ele).val($(ele).attr('oldvalue'));
		        	})
		        	slideOtherDiv(divId,fn);
		        }
		    }).addClass("seriousConfirm");
		}else{
			slideOtherDiv(divId,fn);
		}
	}else{
		slideOtherDiv(divId,fn);
	}
}
	function slideOtherDiv(divId,fn){
		 $.each(slideDivArr,function(index,item){
			if(item.divId == divId){
				if(item.position == 'slideDown'){
					$("#"+item.divId).slideDown(500);
				}else if(item.position == 'bottom'){
					$('#'+item.divId).animate({bottom:'1%'},350,function(){
						try{
							if(typeof fn == 'function') fn();
						}catch(e){}
					});
				}else{
					$('#'+item.divId).animate({right:'0px'},350,function(){
						try{
							if(typeof fn == 'function') fn();
						}catch(e){}
					});
				}
			}else{
				if(item.position == 'slideDown'){
					$("#"+item.divId).slideUp(500);
				}else if(item.position == 'bottom'){
					$('#'+item.divId).animate({bottom:item.value},300);
				}else{
					$('#'+item.divId).animate({right:item.value},300,function(){
						try{
							//if(typeof fn == 'function') fn();
						}catch(e){}
					});
				}
					
			}
		 })
	 }
  
	  //树表格全选半全选
	  /**
	   * 
	   * @param idName:
	   * @param e:事件
	   * @param node:
	   * @param rowText:
	   * @param excludes: 不被控制的项
	   */
	  function accessTree(idName,e,node,rowText,excludes){
			$('#'+idName).css('visibility','hidden');
			e.stopPropagation();
		      var checkedClass = 'tree-checkbox1',
		          middleClass = 'tree-checkbox2',
		          unCheckClass = 'tree-checkbox0';
		      // 叶子或批量只读的节点的id和pid
		      var id = $(node).attr('readid') || $(node).attr('nodeid'),
		      	  pId = $(node).attr('readpid') || $(node).attr('pid'),
		      	  isReadAll = $(node).attr('readid')?true:false;
		      var status = Array.from(node.classList).includes(checkedClass);

		      if(status) {
		    	  var dashbd = $('[nodeid=1]').attr('onclick');
		    	  if(id == 0) {
		    		  if(dashbd) accessClass(node,unCheckClass,excludes);
		    		  else accessClass(node,middleClass,excludes);
		    	  }else accessClass(node,unCheckClass,excludes);
		      }
		      else accessClass(node,checkedClass,excludes);

		      cascade(node,!status,excludes);
		      upAccess(pId);
		      downAccess(id,!status,excludes);
		      // 向上递归
		      function upAccess(pId){
		        if(!pId) return;
		        var isAll = true, has = false,
		            pckbox = $('[readid='+pId+']:visible').get(0);
		        if(pckbox) { // 遍历子节点，子节点的pid属性指向父级id, 排除目录
		            $('[pid='+pId+'],[readpid='+pId+']').not('.tree-dir').each(function(n,item){
		              var clist = Array.from(item.classList),
		                  status = clist.includes(checkedClass);
		              
		              if(!status) isAll = false;
		              else has = true;
		              if(clist.includes(middleClass)) has = true;
		            });

		            if(isAll) accessClass(pckbox,checkedClass,excludes);
		            else if(has)  accessClass(pckbox,middleClass,excludes);
		            else accessClass(pckbox,unCheckClass,excludes);
		            
		            // 级联目录节点状态
					if(!isReadAll){ // 点击叶子节点
						var wckbox = $('[readid='+pId+']').get(0),
							wChecked = Array.from(wckbox.classList).includes(checkedClass),
							rChecked = Array.from(pckbox.classList).includes(checkedClass),
							wmChecked = Array.from(wckbox.classList).includes(middleClass),
							rmChecked = Array.from(pckbox.classList).includes(middleClass);
						var dirnode = $('[nodeid='+pId+']')[0];
						
						if(wChecked && rChecked) { // 批量只读和只写全为勾选时
					    	accessClass(dirnode,checkedClass,excludes);
					    }else if(wChecked || rChecked || wmChecked || rmChecked){
					    	accessClass(dirnode,middleClass,excludes);
					    }else { // 批量只读和只写全为不勾选时
					    	accessClass(dirnode,unCheckClass,excludes);
					    }
					}
		            // 递归处理父级节点状态: 当前节点可能是叶子或批量只读
		            var cpid = $(pckbox).attr('readpid') || $(pckbox).attr('pid');
		            upAccess(cpid);
		        }
		      }
		      // 向下递归
		      function downAccess(id,status,excludes){
		        $('[pid='+id+']').each(function(n,item){
		          var nid = $(item).attr('nodeid');
		          if(status) {
		        	  var wnode = $('[writeid='+nid+']'),
		        		  wChecked = true;
		        	  if(wnode && wnode.length && !wnode.hasClass(checkedClass)) wChecked = false;
		        	  
		        	  // 批量设置只读，当可写为勾选时，节点状态才为勾选，否则为半选
	        		  if(wChecked) accessClass(item,checkedClass,excludes);
	        		  else {
	        			  if(wnode.is(':visible')) accessClass(item,middleClass,excludes);
	        			  else accessClass(item,checkedClass,excludes);
	        		  }
		          }else {
		        	  accessClass(item,unCheckClass,excludes);
		          }
		          
		          if($(item).hasClass('tree-dir')) {// 目录节点
		        	  var rnode = $('[readid='+nid+']'),
		        	  	  rClass = status?checkedClass:unCheckClass;
		        	  //rnode.removeClass(checkedClass).removeClass(unCheckClass).removeClass(middleClass).addClass(rClass);
		        	  accessClass(rnode[0],rClass,excludes);
		          }

		          cascade(item,status,excludes);
		          downAccess($(item).attr('nodeid'),status,excludes);
		        });
		      }
		      function accessClass(node,cls,excludes){
		    	if(excludes) excludes = excludes.split(',');
		    	else excludes = [];
		        if(node && !excludes.includes(node.nextSibling.nodeValue)) {
		          var classes = [checkedClass,middleClass,unCheckClass];
		          classes.map(function(item){
		            if(item == cls) node.classList.add(item);
		            else node.classList.remove(item);
		          });
		        }
		      }
		      // 级联变更
		      function cascade(node,status,excludes){
		        var pTr = $(node).parents('tr:first'),
		            read = pTr.find('.readOper')[0],
		            write = pTr.find('.writeOper')[0],
		            writeChecked = write?Array.from(write.classList).includes(checkedClass):true;
		        var rowText = $(node).parent().text();
		        
		          if(status){ // 只读勾选时
		        	  if(rowText == "EPC"||rowText == "告警确认"||rowText == "Alarm Confirm"||rowText == "告警清除"||rowText == "Clear Alarm"||rowText == "告警恢复"||rowText == "Restore Alarm"||rowText == "告警删除"||rowText == "Delete Alarm"||rowText == "用户向导"||rowText== "User Guide"||rowText == "关于"||rowText == "About"
		        		  ||rowText == "同步"||rowText == "Synchronize"||rowText == "关联的CPEs"||rowText == "CPEs"
								||rowText == "基站IMSI信息清除"||rowText == "Clear IMSI"||rowText == "系统资源监控"||rowText== "Resource"||rowText == "Dashboard"||rowText == "首页"||rowText == "设置"||rowText == "Setting"||rowText == "快速配置"||rowText == "Quick Configuration"||rowText == "激活"||rowText == "Active"||rowText == "网关"||rowText == "eGW"||rowText == "APN"){
		        		  accessClass(read,checkedClass,excludes);
		        		  //accessClass(write,checkedClass,excludes);
		        	  }else{
		        		  accessClass(read,checkedClass,excludes);
		        	  }
	        		  if(!isReadAll) {
	        			  if(!writeChecked) write.click();
	        		  }
		          }else { // 去勾选时
		        	if(!excludes.split(',').includes(rowText)) {
		        		if($(read).attr('readid')==='0') {
		        			var dashbd = $('[nodeid=1]').attr('onclick');
		        			if(dashbd) accessClass(read,unCheckClass,excludes);
		        			else accessClass(read,middleClass,excludes);
		        		}
		        		else accessClass(read,unCheckClass,excludes);
		        	}
		            if(writeChecked && write) write.click();
		            if(!excludes.split(',').includes(rowText)) accessClass(write,unCheckClass,excludes);
		          }
		          // 级联目录节点状态
			      if($(node).attr('readid')){ // 点击批量只读
			    	  var isMiddle = Array.from(node.classList).includes(middleClass);
		    		  var dirNode = $('[nodeid='+id+']');
			    	  if(status || isMiddle || id == 0){
			    		  dirNode.addClass(middleClass).removeClass(unCheckClass);
			    	  }else {
			    		  dirNode.addClass(unCheckClass).removeClass(middleClass).removeClass(checkedClass);
			    	  }
			      }
		      }
		}
	  //后台数据加载完成树表格选中状态
		function review(rootId,ctn){
	        var root = $('[nodeId='+rootId+']',ctn)[0];
	        if(!root) return;
	        var checkedClass = 'tree-checkbox1',
	            middleClass = 'tree-checkbox2',
	            unCheckClass = 'tree-checkbox0';
	        var id = $(root).attr('nodeId');

	        var isAll = true, has = false,
	            children = $('[pId='+id+']',ctn);
	        children.each(function(n,item){
	          review($(item).attr('nodeId'));
	          var clist = Array.from(item.classList),
	              status = clist.includes(checkedClass);
	          if(!status) isAll = false;
	          else has = true;
	          if(clist.includes(middleClass)) has = true;
	        });

	        if(children.length > 0) {
	            if(isAll) accessClass(root,checkedClass);
	            else if(has)  accessClass(root,middleClass);
	            else accessClass(root,unCheckClass);
	        }
	        // 遍历所有可写/只读节点
	        var wChilds = $('[writepid='+id+']',ctn),
	        	rChilds = $('[readviewpid='+id+'],[readpid='+id+']',ctn),
	        	wRoot = $('[writeid='+id+']',ctn)[0],
	        	rRoot = $('[readid='+id+']',ctn)[0],
	        	wStatus = {
	        		isAll: true, 
	        		has: false
		        },
	        	rStatus = {
	        		isAll: true, 
	        		has: false
		        };
	        
	        if(wChilds.length > 0) {// 设置批量可写状态
		        wChilds.each(function(idx,item){
		        	var wlist = Array.from(item.classList),
		                wst = wlist.includes(checkedClass);
		            if(!wst) wStatus.isAll = false;
		            else wStatus.has = true;
		            if(wlist.includes(middleClass)) wStatus.has = true;
		        });
		        
	            if(wStatus.isAll) accessClass(wRoot,checkedClass);
	            else if(wStatus.has)  accessClass(wRoot,middleClass);
	            else accessClass(wRoot,unCheckClass);
	        }
	        if(rChilds.length > 0) {// 设置批量只读状态
	        	rChilds.each(function(idx,item){
		        	var rlist = Array.from(item.classList),
		                rst = rlist.includes(checkedClass);
		            if(!rst) rStatus.isAll = false;
		            else rStatus.has = true;
		            if(rlist.includes(middleClass)) rStatus.has = true;
		        });
		        
	            if(rStatus.isAll) accessClass(rRoot,checkedClass);
	            else if(rStatus.has)  accessClass(rRoot,middleClass);
	            else accessClass(rRoot,unCheckClass);
	        }

	        function accessClass(node,cls){
	          var classes = [checkedClass,middleClass,unCheckClass];
	          classes.map(function(item){
	            if(item == cls) node.classList.add(item);
	            else node.classList.remove(item);
	          });
	        }
	    }
		function proccessData(data){
			data.map(function(item){
	    		if(item.children && item.children.length>0) {
	    			proccessData(item.children);
	    		}else{
	    			delete item.state;
	    		}
	    	});
	    	return data;
		}
  /**
   * 根据有效码集控制元素可见性
   * @param codes: 数组
   **/
   function accessControl(codes){
	   if(!codes) return;
	   var codesRel = {};
	   codes.map(function(code){
		   codesRel[code.key] = code.writable;
	   });
	   codes.map(function(item){
		   if(item.key){
			   var doms = Array.from(document.querySelectorAll('.'+item.key));
	           if(doms.length>0) {
	               doms.map(function(dom){
	            	   var cList = Array.from(dom.classList), writable=false;
	            	   cList.map(function(cItem){/* 多控制时，取合集效果  */
	            		   if(codesRel[cItem]) writable = true;
	            	   });
	            	   setPermission(dom,writable);
	               });
	           }
		   }
	   });
       function setPermission(domObj,visibale){/* 可见设置 */
           var 	cList = Array.from(domObj.classList),
           		isHidden = cList.includes('hidden'),
           		visible = cList.includes('visible');
           if(visibale) {
        	   if(isHidden) domObj.classList.remove('hidden');
           }else {
        	   if(!isHidden) domObj.classList.add('hidden');
        	   if(visible) domObj.classList.remove('hidden');
           }
           
           if(!visible) updateActionStatus(domObj,visibale);
       }
   }
   /**
    * 更新可写操作状态 
    * @param domObj{dom}: 节点元素
    * @param visible{boolean}: 是否可见
    **/
   function updateActionStatus(domObj,visible) {
	   var domVis = domObj.getAttribute('visible');
	   
	   if(domVis != visible) {
		   if(visible) {
			   domObj.removeEventListener('click',disableAction,true);
		   }else {
			   // 捕获阶段拦截点击事件，用于阻止向后响应
			   domObj.addEventListener('click',disableAction,true);
		   }
	   }else {
		   domObj.setAttribute('visible',visible);
	   }
	   
   }
   /**
    * 禁用点击事件 
    * @param evt{event}: 点击事件对象
    **/
   function disableAction(evt) {
	   evt.stopPropagation();
   }
   // 回车表单是否自动提交
   function isAutoSubmit() {
       var userAgent = navigator.userAgent; //取得浏览器的userAgent字符串
       var isIE = userAgent.indexOf("compatible") > -1 && userAgent.indexOf("MSIE") > -1; //判断是否IE<11浏览器
       var isEdge = userAgent.indexOf("Edge") > -1 && !isIE; //判断是否IE的Edge浏览器
       var isIE11 = userAgent.indexOf('Trident') > -1 && userAgent.indexOf("rv:11.0") > -1;
       if(isIE) {
           return false;
       } else if(isEdge) {
           return true;
       } else if(isIE11) {
           return true;
       }else{
           return true;
   	   }
   }
   
   /*<%-- 参数查询结果 --%>*/
   function showGetParamValForRussiaElfcell(data, netType){
   	/*<%-- 将返回来的数据进行组装，并显示；如果处于设置页面，则直接在上方的显示板中显示；如果没有，则在右下角弹出提示来显示结果 --%>*/
   	var divEle = $("<div class='getSetParamValResult'></div>");
       divEle.append($("<span class='resultTitle'>LST RESULT:</span>"));

   	for (var i = 0; i < data.length; i++) {
   		divEle.append($("<br/><span>SmallCell: " + data[i]["cell"] + "</span><br/>"));

   		if (data[i]["paramVals"]) {
   			// 显示参数值
   			divEle.append(createEleByParamValsList(data[i]["paramVals"]));
   		}
   		/*<%-- 显示错误信息 --%>*/
   		if (data[i]["faultCode"]) {
   			divEle.append($("<span>" + CuoWuMa + MaoHao + data[i]["faultCode"] + "</span><br/>"));
   			if(data[i]["faultCode"] == "1001"){
   				divEle.append($("<span>" + CuoWuXinXi + MaoHao + XiaFaMingLingChaoShi + "</span><br/>"));
   			}else{
   				divEle.append($("<span>" + CuoWuXinXi + MaoHao + data[i]["faultString"] + "</span><br/>"));
   			}
   			
   		}
   		//divEle.append($("<hr>"));
   	}

       showAtPromptOrPanel(divEle, netType);
       
       if(!isShowRightClickMenu){
       	 /* 配置结果右键事件 */
       	isShowRightClickMenu = true;
       }
   }
   
   /*<%-- 设置参数的结果 --%>*/
   function showSetParamValForRussiaElfcell(data, objectParam, index, netType) {
       var divEle = $("<div class='getSetParamValResult'></div>");
       divEle.append($("<span class='resultTitle'>MOD RESULT:</span><br/>"));
       for (var i = 0; i < data.length; i++) {
           divEle.append($("<span>SmallCell: " + data[i]["cell"] + "</span><br/>"));
           if (data[i]["paramVals"]) {
           	divEle.append(createEleByParamValsList(data[i]["paramVals"])); 
           }
           if (data[i]["faultCode"]) {
           	divEle.append($("<span>" + CuoWuMa + MaoHao + data[i]["faultCode"] + "</span><br/>"));
               if(data[i]["faultCode"]=="1001"){
               	divEle.append($("<span>" + CuoWuXinXi + MaoHao + XiaFaMingLingChaoShi + "</span><br/>"));
               }else{
               	divEle.append($("<span>" + CuoWuXinXi + MaoHao + data[i]["faultString"] + "</span><br/>"));
               }
           }
           //divEle.append($("<hr>"));
       }
       showAtPromptOrPanel(divEle, netType);
       
       if(!isShowRightClickMenu){
      	 /* 配置结果右键事件 */
      	 isShowRightClickMenu = true;
       }
   }
	//定时刷新任务列表
    function refreshTasklist(taskListId,url,typeId){
		if(typeId){
			var taskType = "";
				try{
					taskType = $("#"+typeId).combobox("getValue");
				} catch(e){
			}
		}
		var tableTaskList = $("#"+taskListId);
		if (tableTaskList.length > 0) {
			var pageNumber = $("#"+taskListId).datagrid('options').pageNumber;
			var pageSize = $("#"+taskListId).datagrid('options').pageSize;
			var param = {};
			if(typeId){
				param ={
					timeZone:timeZone,
					page:pageNumber,
					rows:pageSize,
					taskType : taskType,
					cpeType : taskType,
					taskName :taskSearchText,
					startTime : queryStartTime,
					endTime : queryEndTime
				}
			}else{
				param ={
					timeZone:timeZone,
					page:pageNumber,
					rows:pageSize,
					likeFields :"task_name",
					searchText: taskSearchText,
					startTime : queryStartTime,
					endTime : queryEndTime
				}
			}
			var selectBefore = "";
			try{
				selectBefore = $("#"+taskListId).datagrid('getSelected');
			}catch(e){
				
			}
			var selectRowTaskId = "";
			if(selectBefore!=null){
				selectRowTaskId = selectBefore.TASK_ID;
			}
			$.post(url, param,function (data) {
				$.each(data.rows,function(index,item){
					if(item.TASK_ID == selectRowTaskId){
						$("#"+taskListId).datagrid('selectRow',index);
					}
					var index = $("#"+taskListId).datagrid('getRowIndex',item.TASK_ID);
					if(index > -1) {
						$("#"+taskListId).datagrid('updateRow',{
							index:index,
							row:item
						})
					}
				})
		    }, "json");
		}
	}
	//定时刷新结果列表
	function refreshReaultlist(resultListId,url,taskTableId){
		if(taskTableId){
			var selectedTask = $("#"+taskTableId).datagrid("getSelected");
			var task_type =  selectedTask["TYPE"];
		}
		var tableResultList = $("#"+resultListId);
		if (tableResultList.length > 0) {
			var param = {};
			if(taskTableId){
				param ={
					timeZone:timeZone,
					type: task_type,
					searchText: resultSearchText
				}
			}else{
				param ={
					timeZone:timeZone,
					searchText: resultSearchText
				}
			}
			var selectBefore = "";
			try{
				selectBefore = $("#"+resultListId).datagrid('getSelected');
			}catch(e){
				
			}
			if(selectBefore){
				var selectRowId = selectBefore.SERIAL_NUMBER;
			}
			$.post(url, param,function (data) {
				if(data.grid){
					$("#"+resultListId).datagrid('loadData',data.grid);
					if(selectBefore){
						$.each(data.grid.rows,function(index,item){
							if(item.SERIAL_NUMBER == selectRowId){
								$("#"+resultListId).datagrid('selectRow',index);
							}
						} )
					}
				}
		    }, "json");
		}
	}
	
	//任务列表     模糊匹配按钮点击
	function vagueTaskSearchFun(TaskListId,TaskListSearchId,startTimeId,endTimeId){
		taskSearchText = $('#'+TaskListSearchId).val();
		queryStartTime = "";
		queryEndTime = "";
		$("#"+startTimeId).datetimebox('setValue', null);
		$("#"+endTimeId).datetimebox('setValue', null);
		$('#'+TaskListId).datagrid('reload');
	}
	//任务列表   精确匹配按钮点击
	function accurateTaskSearchFun(TaskListId,TaskListSearchId,startTimeId,endTimeId){
		taskSearchText = "";
		queryStartTime = $('#'+startTimeId).datetimebox("getValue");
		queryEndTime = $('#'+endTimeId).datetimebox("getValue");
		var validTimeResult = validateStartAndStopTime(queryStartTime, queryEndTime);
	    if ("false" == validTimeResult) {
	    	showMsg("prompt_msg",JieShuShiJianBuNengXiaoYuKaiShiShiJian);
	        return;
	    }
		$("#"+TaskListSearchId).val("");
		$('#'+TaskListId).datagrid('reload');
	}
	//任务列表    重置按钮点击
	function taskResetQueryInput(TaskListSearchId,startTimeId,endTimeId){
		//$("#"+TaskListSearchId).val("");
		$("#"+startTimeId).datetimebox('setValue', null);
		$("#"+endTimeId).datetimebox('setValue', null);
	}
    //任务状态格式化：0-等待  1-进行中  2-已结束  3-终止中
	function taskTableStatus(value, rowData, rowIndex) {
		if (value == "1") {
			return '<span class="el-icon el-icon-status-waiting1" style="margin-right:5px;"></span> '+DengDai;
		} else if (value == "2") {
			return '<span class="el-icon el-icon-status-inProgress"  style="margin-right:5px;"></span> '+JinXingZhong;
		} else if (value == "3" || value=="6") {
			return '<span class="el-icon el-icon-status-suspend"  style="margin-right:5px;"></span> '+ZanTing;
		} else if (value == "4") {
			return '<span class="el-icon el-icon-status-terminate"  style="margin-right:5px;"></span> '+YiJieShu;
		} else if (value == "5") {
			return '<span class="el-icon el-icon-status-inProgress"  style="margin-right:5px;"></span> '+JinXingZhong;
		} else {
			return "";
		}
	}
	//备份与恢复 - task list 状态格式
	function backupRestoreTaskTableStatus(value, rowData, rowIndex) {
		if (value == "1") {
			return '<span class="el-icon el-icon-status-waiting1" style="margin-right:5px;"></span> '+DengDai;
		} else if (value == "2") {
			return '<span class="el-icon el-icon-status-inProgress"  style="margin-right:5px;"></span> '+JinXingZhong;
		} else if (value == "3" || value=="6") {
			return '<span class="el-icon el-icon-status-suspend"  style="margin-right:5px;"></span> '+ZanTing;
		} else if (value == "4") {
			return '<span class="el-icon el-icon-status-terminate"  style="margin-right:5px;"></span> '+YiJieShu;
		} else if (value == "5") {
			return '<span class="el-icon el-icon-status-terminate"  style="margin-right:5px;"></span> '+YiJieShu;
		} else {
			return "";
		}
	}
	//修改密码 - task list 状态格式
	function changePasswordTaskTableStatus(value, rowData, rowIndex) {
		if (value == "1") {
			return '<span class="el-icon el-icon-status-waiting1" style="margin-right:5px;"></span> '+DengDai;
		} else if (value == "2") {
			return '<span class="el-icon el-icon-status-inProgress"  style="margin-right:5px;"></span> '+JinXingZhong;
		} else if (value == "3" || value=="6") {
			return '<span class="el-icon el-icon-status-suspend"  style="margin-right:5px;"></span> '+ZanTing;
		} else if (value == "4") {
			return '<span class="el-icon el-icon-status-terminate"  style="margin-right:5px;"></span> '+YiJieShu;
		} else if (value == "5") {
			return '<span class="el-icon el-icon-status-terminate"  style="margin-right:5px;"></span> '+YiJieShu;
		} else {
			return "";
		}
	}
  //cpe 设备上报日志 - task list 状态格式
	function cpeLogTaskTableStatus(value, rowData, rowIndex) {
		if (value == "0") {
			return '<span class="el-icon el-icon-status-waiting1" style="margin-right:5px;"></span> '+DengDai;
		} else if (value == "1") {
			return '<span class="el-icon el-icon-status-inProgress"  style="margin-right:5px;"></span> '+JinXingZhong;
		} else if (value == "2") {
			return '<span class="el-icon el-icon-status-terminate"  style="margin-right:5px;"></span> '+YiJieShu;
		} else if (value == "3") {
			return '<span class="el-icon el-icon-status-terminate"  style="margin-right:5px;"></span> '+YiJieShu;
		} else if (value == "4") {
			return '<span class="el-icon el-icon-status-terminate"  style="margin-right:5px;"></span> '+ZhongZhi;
		} else {
			return "";
		}
	}
	//任务执行结果格式化
	function taskTableResult(value, rowData, rowIndex) {
		if (value == "1") {
			return ChengGong;
		} else if (value == "2") {
			return BuFenChengGong;
		} else if (value == "3") {
			return ShiBai;
		} else {
			return "";
		}
	}
   function resultTableStatus(value, rowData, rowIndex){//5-升级超时后的升级等待
   		if (value == "1") {
			return '<span class="el-icon el-icon-status-waiting1" style="margin-right:5px;"></span> '+DengDai;
		} else if (value == "2" || value == "5") {
			return '<span class="el-icon el-icon-status-inProgress"  style="margin-right:5px;"></span> '+JinXingZhong;
		} else if (value == "3") {
			return '<span class="el-icon el-icon-status-suspend"  style="margin-right:5px;"></span> '+ZanTing;
		} else if (value == "4") {
			return '<span class="el-icon el-icon-status-terminate"  style="margin-right:5px;"></span> '+YiJieShu;
		} else {
			return "";
		}
   }
   function resultTableResult(value, rowData, rowIndex){
   		if (value == "3") {
			return ShiBai;
		} else if (value == "1") {
			return ChengGong;
		} else if (value == "2") {
			return ZhongZhi;
		}else {
			return "";
		}
   }
   /**
    * 生成过滤菜单
    * @data: 菜单列表数据
    * @fn: 回调函数
    * @click: 行点击事件
    */
   function filterMenu({data,fn,click}) {
     $('.filter-menu').remove();
     var tips = $('<div class="filter-menu"></div>');
   	 if(data && data.length == 0) return;
    
     var  ckAllCtn = $('<div class="filter-item" style="display:flex;align-items: center;padding: 3px;"></div>'),
          ckAll = $('<input type="checkbox" checked/>');
     ckAllCtn.append(ckAll).append('<label style="padding-left: 7px;white-space: nowrap;">All</label>');
     tips.append(ckAllCtn);

     ckAll.on('click',function(){
       var _self = this,
           bool = $(this).is(':checked'),
           ckItems = $('input[name]',tips),
           ckVals = [];

       ckItems.each(function(idx,item){
           if(bool || idx == 0) {
             ckVals.push(item.value);
             item.checked = true;
             if(ckItems.length==1) {
               ckAll.prop('checked',isAllCk);
             }
           }else {
             item.checked = false;
             _self.indeterminate = true;
           }
       });
       
       bool = $(this).is(':checked')
       if(bool) {
        	$('.icon-filter.clicked').removeClass('selected');
       }else {
        	$('.icon-filter.clicked').addClass('selected');
       }
       
       if(click){
           try { click(ckVals.join(',')); } catch (e) {}
       }
     })

     data.map(function(item){
       var itemDiv = $('<div class="filter-item" style="display:flex;align-items: center;padding: 3px;"></div>'),
           itemVal = item.value ,
           rdId = Math.random().toString(32).substring(2),
           ck = $('<input id="'+rdId+'" type="checkbox" name="'+item.name+'" value="'+itemVal+'"/>'),
           label = $('<label for="'+rdId+'" style="padding-left: 7px;white-space: nowrap;">'+item.label+'</label>');

       if(item.checked==true){
    	   ck.prop('checked',true);
       }
       if(item.disabled==true){
    	   ck.prop('disabled',true);
       }

       ck.off('click').on('click',function(){
         var values = '';
         $('[name="'+item.name+'"]:checked').each(function(index,ckItem){
           values += ','+$(ckItem).val();
         });
         
         var isStop = false;
         if(values) values = values.substring(1);
         else {
        	 values = $(this).val();
        	 this.checked = true;
        	 isStop = true;
         }
         
         var iptTotal = $('[name="'+item.name+'"]',tips).length,
         	ckNum = $('[name="'+item.name+'"]:checked',tips).length;
         
         if(iptTotal>ckNum){
        	 $('.icon-filter.clicked').addClass('selected');
           ckAll.prop('checked',false);
           ckAll.prop('indeterminate',true);
         }else{
        	 $('.icon-filter.clicked').removeClass('selected');
           ckAll.prop('checked',true);
           ckAll.prop('indeterminate',false);
         }
         if(isStop) return;
         
         if(click){
           try { click(values); } catch (e) {}
         }
       });
       tips.append(itemDiv.append(ck).append(label));
     });
     // 设置全选状态
     var iptTotal = $('input[name]',tips).length,
         ckNum = $('input[name]:checked',tips).length,
         isAllCk = iptTotal==ckNum;
     ckAll.prop('checked',isAllCk);
     ckAll.prop('indeterminate',!isAllCk);

     if(fn){
       try { fn(tips); } catch (e) {}
     }
   }
   /**
    * 表头过滤
    * 依赖于全局变量rules: [{match:function(code){},action:function(code){},title:'xxx',checked:false},...]
    * match: 匹配规则
    * action: 表头过滤点击事件
    * title: 表头要展示的文字
    */
   function titleFilter(){
     var span = $('<span class="icon-filter filter-opt"></span>'), title = $(this),
         filterRules = [],
         code = title.parents('td:first').attr('field');
     try {
       filterRules = rules;
     } catch (e) {}
     title.after(span);

     span.off('click').on('click',function(){
       $('.icon-filter').removeClass('clicked');
       $(this).addClass('clicked');
       try{
         filterRules.map(function(rule){
           if(rule.match(code)) {
        	   rule.action(code,event);
           }
         });
       }catch(e){}
       event.stopPropagation();
     });

     var titleText = '';
     try {
       filterRules.map(function(rule){
         if(rule.match && rule.match(code)) {
        	 titleText = rule.title;
         }
       });
       return titleText;
     } catch (e) {}

     return titleText;
   }
   /* easyui datagrid rownumber width */
   $.extend($.fn.datagrid.methods, {
	    fixRownumber : function (jq) {
	        return jq.each(function () {
	            var panel = $(this).datagrid("getPanel");
	            //获取最后一行的number容器,并拷贝一份
	            var clone = $(".datagrid-cell-rownumber", panel).last().clone(); //由于在某些浏览器里面,是不支持获取隐藏元素的宽度,所以取巧一下
	            clone.css({
	                "position" : "absolute",
	                left : -1000
	            }).appendTo("body");
	            var width = clone.width("auto").width();
	            //默认宽度是25,所以只有大于25的时候才进行fix
	            if (width > 25) {
	                //多加5个像素,保持一点边距
	                $(".datagrid-header-rownumber,.datagrid-cell-rownumber", panel).width(width + 5);
	                //修改了宽度之后,需要对容器进行重新计算,所以调用resize
	                $(this).datagrid("resize");
	                //一些清理工作
	                clone.remove();
	                clone = null;
	            } else {
	                //还原成默认状态
	                $(".datagrid-header-rownumber,.datagrid-cell-rownumber", panel).removeAttr("style");
	            }
	        });
	    }
	});
   //数组删除某个元素
   Array.prototype.indexOf = function(val){
	   for(var i=0;i<this.length;i++){
		   if(this[i] == val){
			   return i;
		   }
	   }
	   return -1;
   }
   Array.prototype.removeArrElement = function(val){
	   var index = this.indexOf(val);
	   if(index > -1){
		   this.splice(index,1);
	   }
   }
  /* eg:arr.removeArrElement("");*/
 //数组删除某个元素  结束

   /* 频点转频率 */
   function translateToFre(e){
   	if(typeof(e) === "string"){
   		var EARFCN = e.trim();
   		if(e == "" || e == " "){
   			return "";
   		}
   	}else{
   		var EARFCN = $(e).val()||"";
   		if(EARFCN.indexOf("(")!= -1){
   			EARFCN = EARFCN.split("(")[0];
   		}
   	}
   	var reg = /^\+?[1-9][0-9]*$/;
   	
   	if(reg.test(EARFCN)){
   	//var EARFCN = value;//正常显示的频点值  还需要将此值转换成频率
   		var frequency = 0;
   		if (EARFCN >= 36000 && EARFCN <= 36199) { //tdd-band 33
   			frequency = 1900 + 0.1 * (EARFCN - 36000);
   		} else if (EARFCN >= 36200 && EARFCN <= 36349) { //tdd-band 34
   			frequency = 2010 + 0.1 * (EARFCN - 36200);
   		} else if (EARFCN >= 36350 && EARFCN <= 36949) { //tdd-band 35
   			frequency = 1850 + 0.1 * (EARFCN - 36350);
   		} else if (EARFCN >= 36950 && EARFCN <= 37549) { //tdd-band 36
   			frequency = 1930 + 0.1 * (EARFCN - 36950);
   		} else if (EARFCN >= 37550 && EARFCN <= 37749) { //tdd-band 37
   			frequency = 1910 + 0.1 * (EARFCN - 37550);
   		} else if (EARFCN >= 37750 && EARFCN <= 38249) { //tdd-band 38
   			frequency = 2570 + 0.1 * (EARFCN - 37750);
   		} else if (EARFCN >= 38250 && EARFCN <= 38649) { //tdd-band 39
   			frequency = 1880 + 0.1 * (EARFCN - 38250);
   		} else if (EARFCN >= 38650 && EARFCN <= 39649) { //tdd-band 40
   			frequency = 2300 + 0.1 * (EARFCN - 38650);
   		} else if (EARFCN >= 39650 && EARFCN <= 41589) { //tdd-band 41
   			frequency = 2496 + 0.1 * (EARFCN - 39650);
   		} else if (EARFCN >= 41590 && EARFCN <= 43589) { //tdd-band 42
   			frequency = 3400 + 0.1 * (EARFCN - 41590);
   		} else if (EARFCN >= 43590 && EARFCN <= 45589) { //tdd-band 43
   			frequency = 3600 + 0.1 * (EARFCN - 43590);
   		} else if (EARFCN >= 18000 && EARFCN <= 18599) { //fdd-band 1
   			frequency = 1920 + 0.1 * (EARFCN - 18000);
   		} else if (EARFCN >= 0 && EARFCN <= 599) {
   			frequency = 2110 + 0.1 * (EARFCN - 0);
   		} else if (EARFCN >= 18600 && EARFCN <= 19199) { //fdd-band 2
   			frequency = 1850 + 0.1 * (EARFCN - 18600);
   		} else if (EARFCN >= 600 && EARFCN <= 1199) {
   			frequency = 1930 + 0.1 * (EARFCN - 600);
   		} else if (EARFCN >= 19200 && EARFCN <= 19949) { //fdd-band 3
   			frequency = 1710 + 0.1 * (EARFCN - 19200);
   		} else if (EARFCN >= 1200 && EARFCN <= 1949) {
   			frequency = 1805 + 0.1 * (EARFCN - 1200);
   		} else if (EARFCN >= 19950 && EARFCN <= 20399) { //fdd-band 4
   			frequency = 1710 + 0.1 * (EARFCN - 19950);
   		} else if (EARFCN >= 1950 && EARFCN <= 2399) {
   			frequency = 2110 + 0.1 * (EARFCN - 1950);
   		} else if (EARFCN >= 20400 && EARFCN <= 20649) { //fdd-band 5
   			frequency = 824 + 0.1 * (EARFCN - 20400);
   		} else if (EARFCN >= 2400 && EARFCN <= 2649) {
   			frequency = 869 + 0.1 * (EARFCN - 2400);
   		} else if (EARFCN >= 20650 && EARFCN <= 20749) { //fdd-band 6
   			frequency = 830 + 0.1 * (EARFCN - 20650);
   		} else if (EARFCN >= 2650 && EARFCN <= 2749) {
   			frequency = 875 + 0.1 * (EARFCN - 2650);
   		} else if (EARFCN >= 20750 && EARFCN <= 21449) { //fdd-band 7
   			frequency = 2500 + 0.1 * (EARFCN - 20750);
   		} else if (EARFCN >= 2750 && EARFCN <= 3449) { 
   			frequency = 2620 + 0.1 * (EARFCN - 2750);
   		} else if (EARFCN >= 3450 && EARFCN <= 3799) { //fdd-band 8
   			frequency = 925 + 0.1 * (EARFCN - 3450);
   		} else if (EARFCN >= 5010 && EARFCN <= 5179) { //fdd-band 12
   			frequency = 729 + 0.1 * (EARFCN - 5010);
   		} else if (EARFCN >= 5180 && EARFCN <= 5279) { //fdd-band 13
   			frequency = 746 + 0.1 * (EARFCN - 5180);
   		} else if (EARFCN >= 5730 && EARFCN <= 5849) { //fdd-band 17
   			frequency = 734 + 0.1 * (EARFCN - 5730);
   		} else if (EARFCN >= 6150 && EARFCN <= 6449) { //fdd-band 20
   			frequency = 791 + 0.1 * (EARFCN - 6150);
   		}else if (EARFCN >= 9210 && EARFCN <= 9659) { //fdd-band 28       758-803 9210-9659
   			frequency = 758 + 0.1 * (EARFCN - 9210);
   		}else if(EARFCN >= 55240 && EARFCN <= 56740){
   			frequency = 3550 + 0.1*(EARFCN - 55240);
   		}else if(EARFCN >= 46790 && EARFCN <= 54539){
   			frequency = 5150 + 0.1*(EARFCN - 46790);
   		}else if(EARFCN >= 63000 && EARFCN <= 63999){
   			frequency = 5150 + 0.1*(EARFCN - 63000);
   		}else if(EARFCN >= 64000 && EARFCN <= 64999){
   			frequency = 5725 + 0.1*(EARFCN - 64000);
   		}else if(EARFCN >= 46790 && EARFCN <= 54539){
   			frequency = 5150 + 0.1*(EARFCN - 46790);
   		}else if(EARFCN >= 63000 && EARFCN <= 63999){
   			frequency = 5150 + 0.1*(EARFCN - 63000);
   		}else if(EARFCN >= 64000 && EARFCN <= 64999){
   			frequency = 5725 + 0.1*(EARFCN - 64000);
   		}else {
   			//throw new Exception("Please Input the right EARFCN!");
   			frequency = "--";
   		}
   		var showStr = EARFCN.toString()+"("+frequency.toString()+"MHz"+")";
   		$(e).val(showStr);
   		return showStr;
   	}else{
   		return false;
   	}
   }
   function translateToEarfen(e){
   	var allvalue = $(e).val();
   	var trueFlag = allvalue.indexOf("(");
   	if(trueFlag > -1){
   		var valueArr = allvalue.split("(");
   		allvalue = valueArr[0];
   	}
   	$(e).val(allvalue);
   }
   
   /** 
    * 扩展datagrid api 
    * setRow的参数格式：{idField: '',usable: false},如：grid.datagrid('setRow',{id:'SN_012389',usable: false});
    * checkUnable的参数格式：直接传row即可
    * initRow：没有参数
    * **/
   $.extend($.fn.datagrid.methods, {
       setRow : function (jq,params) {
           return jq.each(function () {
               var grid = $(this),
                   ops = grid.datagrid('options'),
                   idField = ops.idField,
                   unUsableIds = grid.data('idFields')||[];
               if(idField){
                 var index = grid.datagrid('getRowIndex',params[idField]);
                 var tr = grid.datagrid('getPanel').find('.datagrid-btable tr[datagrid-row-index="'+index+'"]');

                 if(params.usable == false) {
                   tr.find('.datagrid-cell-check').css({visibility:'hidden'});
                   if(!unUsableIds.includes(params[idField]))  unUsableIds.push(params[idField]);
                   if(index >= 0) grid.datagrid('unselectRow',index);
                 }else{
                   tr.find('.datagrid-cell-check').css({visibility:'visible'});
                   var kdx = unUsableIds.indexOf(params[idField]);
                   unUsableIds.splice(kdx,1);
                 }
                 grid.data('idFields',unUsableIds);
               }
           });
       },
       initRow : function(jq,params){
         return jq.each(function () {
           var grid = $(this),
               ids = grid.data('idFields')||[],
               opts = grid.datagrid('options');
           ids.map(function(item){
             var param = {usable:false};
             param[opts.idField] = item;
             grid.datagrid('setRow',param);
           });
         });
       },
       checkUnable : function(jq,params){
         var grid = $(jq),
             ids = grid.data('idFields') || [],
             opts = grid.datagrid('options'),
             bool = ids.includes(params[opts.idField]);
         if(bool){
           setTimeout(function(){
             var index = grid.datagrid('getRowIndex',params[opts.idField]);
             grid.datagrid('unselectRow',index);
           },0);
         }
         return bool;
       }
   });
   /******************** domRender start ********************/
   /* 是否数组类型 */
   function isArray(o) {
     return Object.prototype.toString.call(o) === '[object Array]';
   }
   function openPropsPanel(bool){
     var ctner = document.querySelector('.form-ctn');
     if (bool == false){
       $('.form-operations').css({display:'block'});
       $('.form_bt_reback').removeClass('show');
       $('.form_bt_refresh').show();
       $('.form-ctn .form-props').removeClass('show').css({top: '-100%'}).html('');
       $(ctner).css({overflow:'auto'});
       try{
         eNbSetting.changeTitle(false);
       }catch(e){}
     } else{
       $('.form-operations').css({display:'none'});
       $('.form_bt_reback').addClass('show');
       $('.form_bt_refresh').hide();
       $('.form-ctn .form-props').addClass('show').css({top: ctner.scrollTop+'px'});
       $(ctner).css({overflow:'hidden'});
     }
   }
   /* 渲染二级编辑页面 */
   function propsRender(columns, ctner, title) {
     var pDiv = ctner.querySelector('.form-props'),
       wDiv = document.createElement('form'),
       bDiv = document.createElement('div');
     pDiv.innerHTML = '';
     wDiv.classList.add('form-wrap');
     wDiv.style.setProperty('overflow-y', 'auto');
     wDiv.style.setProperty('overflow-x', 'hidden');

     openPropsPanel();
     try {
       eNbSetting.changeTitle(title); /* 这里的代码需要优化成更灵活的方式 */
     } catch (e) {}

     // 按钮处理
     var bt_div = document.createElement('div'),
       bt_ok = document.createElement('a'),
       bt_no = document.createElement('a'),
       span_ok = document.createElement('span'),
       span_no = document.createElement('span');
     bt_ok.classList.add('linkbutton');
     bt_no.classList.add('linkbutton');
     bt_no.classList.add('linkbutton_nowanna');
     span_ok.innerText = Render.propOkTxt;
     span_no.innerText = Render.propCancelTxt;
     bt_ok.appendChild(span_ok);
     bt_no.appendChild(span_no);
     bt_no.addEventListener('click', function(event) {
       openPropsPanel(false);
       try {
         eNbSetting.changeTitle(false);
       } catch (e) {}
     });
     bt_div.appendChild(bt_ok);
     bt_div.appendChild(bt_no);
     bt_div.classList.add('bt-group');
     bt_div.classList.add('linkbuttonGroup');
     bDiv.style.setProperty('padding-left', '30px');
     bDiv.style.setProperty('padding-bottom', '10px');
     bDiv.appendChild(bt_div);

     pDiv.appendChild(wDiv);
     pDiv.appendChild(bDiv);
     domRender(columns, wDiv, true);
   }
   /* 分组渲染 */
   function groupRender(groups, ctner) {
	 Render.validResult.reboot = false;
     Render.originalGroups = groups;
     Render.originalCtner = ctner;
     Render.tableCollector = {};
     $('.form_bt_reback').removeClass('show');

     if (!isArray(groups)) groups = [groups];
     var navs = [],
         fragment = document.createDocumentFragment();
     // 迭代依次渲染各组节点
     groups.map(function(group, index) {
       var rd = 'ar_' + Math.random().toString(32).substring(2, 6);
       navs.push({
         title: group.title,
         id: rd
       });
       var gPanel = _initGroupPanel(group, rd);
       if (group.groups) {
         group.groups.map(function(item) {
           if (group.groups.length > 1) {
             var subTitle = document.createElement('div'); // 子集分类标题
             //4860 邻区 临频 模块，页面增重启生效提示
             if(['Intel_CR_CA','Intel_CR_SC','Intel_CR_TC','Intel_CR_DC','Intel_CR','MLN','MLN_CA','MLN_SC','MLN_DC'].includes(enbPlatformType)){
              if(['Neigh Freq','Neigh Cell','邻频','邻区',null,undefined].includes(item.title)){
                subTitle.innerText = '. ' + (item.title || '') + '\xa0\xa0\xa0\xa0' + rebootText;
              }else{
                subTitle.innerText = '. ' + (item.title || '');
              }
            }else{
              subTitle.innerText = '. ' + (item.title || '');
             }
             subTitle.classList.add('form-single');
             subTitle.classList.add('group-sub-title');
             gPanel.appendChild(subTitle);
           }
           domRender(item.list, gPanel);
         });
       }

       if (index == groups.length - 1) gPanel.parentNode.classList.add('last');
     });
     ctner.appendChild(fragment);
     // 属性设置层
     var props = document.createElement('div');
     props.classList.add('form-props');
     ctner.appendChild(props);

     // 按钮处理
     var bts_exist = $('.form-operations');
     if (bts_exist.length > 0) bts_exist.remove();
     var bDiv = document.createElement('div'),
       bt_div = document.createElement('div'),
       bt_ok = document.createElement('a'),
       bt_no = document.createElement('a'),
       span_ok = document.createElement('span'),
       span_no = document.createElement('span');
     bt_ok.classList.add('linkbutton');
     bt_no.classList.add('linkbutton');
     bt_no.classList.add('linkbutton_nowanna');
     span_ok.innerText = Render.submitTxt;
     span_no.innerText = Render.resetTxt;
     bt_ok.addEventListener('click', function() {
       Render.submit();
     });
     bt_no.addEventListener('click', function() {
       Render.reset();
     });
     bt_ok.appendChild(span_ok);
     bt_no.appendChild(span_no);
     bt_div.appendChild(bt_ok);
     bt_div.appendChild(bt_no);
     bt_div.classList.add('bt-group');
     bt_div.classList.add('linkbuttonGroup');
     bDiv.classList.add('form-operations');
     bDiv.classList.add('el-card__footer');
   /*  bDiv.style.setProperty('padding-left', '40px');
     bDiv.style.setProperty('padding-bottom', '10px');*/
     bDiv.appendChild(bt_div);

     $(bDiv).append('<div class="success">' + Render.status.success + '</div>');

     $(ctner).after(bDiv);

     //_navRender(navs, ctner)
       // 初始化导航
     function _navRender(navs, ctner) {
       if (navs && navs.length > 1) {
         var slide = document.createElement('div');
         navs.map(function(nav, idx) {
           var item = document.createElement('a');
           if (idx == 0) $(item).addClass('selected')
           item.innerText = nav.title;
           $(item).data('data', nav);
           item.addEventListener('click', function() { /* 导航跳转 */
             var data = $(this).data('data'),
               pT = document.querySelector('#' + data.id).parentNode,
               cls = pT.classList;
             if (Array.from(cls).includes('extend')) {
               $(pT).find('.title-text').click();
             }
             setTimeout(function() {
               ctner.scrollTop = pT.offsetTop - 5;
             }, 500);
             $(this).addClass('selected').siblings().removeClass(
               'selected');
           });
           slide.appendChild(item);
         });
         slide.classList.add('form-slide-nav');
         ctner.appendChild(slide);
         ctner.addEventListener('scroll', function(event) {
           slide.style.setProperty('top', ctner.scrollTop + 100 + 'px');
         });
       }
     }
     // 初始化分组面板 并返回
     function _initGroupPanel(grp, titleId) {
       var panel = document.createElement('div'),
         pTitle = document.createElement('div'),
         titleIcon = document.createElement('span'),
         titleSp = document.createElement('span'),
         pBody = document.createElement('div');

       titleIcon.classList.add('title-icon');

       titleSp.classList.add('title-text');
       titleSp.innerText = grp.title || ' ';
       titleSp.addEventListener('click', function() {
         var clss = this.parentNode.parentNode.classList;
         if (Array.from(clss).includes('extend')) {
           clss.remove('extend');
           $(pBody).slideDown(500, function() {
             $(window).resize();
             Array.from($('table[id]', panel)).map(function(item){
              $(item).datagrid();
             });
           });
         } else {
           clss.add('extend');
           $(pBody).slideUp();
         }
       });

       pTitle.classList.add('group-title');
       pTitle.setAttribute('id', titleId);
       pTitle.style.marginLeft = '20px';
       pTitle.appendChild(titleIcon);
       pTitle.appendChild(titleSp);

       pBody.classList.add('form-wrap');
       panel.classList.add('form-group');
       panel.appendChild(pTitle);
       panel.appendChild(pBody);
       //ctner.appendChild(panel);
       fragment.appendChild(panel);
       return pBody;
     }
   }
   /**
    * dom渲染器
    * @param datas: 节点的属性数据（对象数组）
    * @param ctner: 节点放置的容器对象（dom、默认是document.body）
    * @param isSub: 是否为明显层渲染
    **/
   function domRender(datas, ctner, isSub) {
     ctner = ctner || document.body;

     _dataToDom();
     /*---------------------以下皆为公共处理封装---------------------*/
     /* 对象数组排序 */
     function _dataSort(objA, objB) {
       var val1 = objA.order || '',
         val2 = objB.order || '';
       if (val1 < val2) return -1;
       else if (val1 > val2) return 1;
       else return 0;
     }
     /**
      * 数据映射为dom节点
      **/
     function _dataToDom() {
       if (datas) {
         //datas = datas.sort(_dataSort);
         // 支持解析的组件类型
         var clsKeys = $.domRenderDefaults;
         datas.map(function(item) { // 这种生成dom方式可能会有性能问题，可以考虑documentFragment方式
           var div = document.createElement('div'),
             cKey = item.type || 'text',
             input = document.createElement('input');
           $(input).data('option', item); // 用于校验方法
           input.classList.add('easyui-' + clsKeys[cKey]); // 添加对应组件class，用于自动化解析

           _initProps(input, item); // 属性初始化
           _combinFormItem(div, input, item); // 表单单元合成

           (function(domVal, domProps) {
             setTimeout(function() {
               _parseCascade(domVal, domProps, true);
               var fn = $(input).data('onchange');
               if (fn) {
                   if (typeof fn == 'function') fn(domVal, '', domProps);
                   else eval(fn)(domVal, '', domProps);
               }
               if (domProps.suffix) {
                 $(input).next().find('input').blur();
               }
             },0);
           })(item.value, item);

           $(ctner).append(div);
         });
         $.parser.parse(ctner); // 解析生成的组件节点
       }
     }
     /**
      * 表单单元合成
      **/
     function _combinFormItem(div, input, item) {
       var clsKeys = $.domRenderDefaults,
         cKey = item.type || 'text',
         label = document.createElement('label'),
         input4range = document.createElement('input'), // 用于双输入域类型
         wrapDiv = document.createElement('div');
       wrapDiv.classList.add('form-item-wrap');

       if (cKey != 'list') { // 常规输入域
         label.classList.add('form-title');
         label.innerText = item.label;
         div.appendChild(label);

         $(input4range).data('option', item);
         input4range.classList.add('easyui-' + clsKeys[cKey]);
         wrapDiv.appendChild(input);
         // 处理单双输入域
         if (cKey == 'range') {
           if (!item.button) _initProps(input4range, item);
           $(wrapDiv).append(' - ');
           wrapDiv.appendChild(input4range);
         } else {
           wrapDiv.appendChild(input);
         }
         
         if(cKey == 'bind') {
           if (!item.button) {
              var select4bind = document.createElement('select'),
                  bindData = item.bindObj?item.bindObj.data:[];
              
              select4bind.id = item.name + '_sufix';
              bindData.map(function(item){
                  var op = document.createElement('option');

                  op.value = item;
                  op.innerText = item;
                  select4bind.appendChild(op);
              });

              wrapDiv.appendChild(select4bind);
           }
         }

         // 处理按钮
         if (item.button) { // 如果有按钮项，自动追加隐藏域 显示域的name后缀加  '_show'
           var _showProps = $.extend({}, item, {
                  value: '',
                  onChange: '',
                  reboot: 0
                });

           _showProps.name = _showProps.name + '_show';
           $(input4range).data('option', _showProps);
           $(input).data('fns', []); // 清除隐藏的blur校验

           _initProps(input4range, _showProps);
           wrapDiv.appendChild(input4range);

           if (cKey == 'range') {
             var cloneName = $(input4range).attr('name'),
                 cloneDom = $(input4range).clone(true);
             cloneDom.attr('name', cloneName + '_left').attr('id', cloneName + '_left');
             $(input).after(cloneDom);
           }

           if(cKey == 'bind') {
              var select4bind = document.createElement('select'),
                  labeldom = document.createElement('label'),
                  bindObj = item.bindObj || {},
                  labelTitle = bindObj.title||'',
                  bindData = bindObj.data||[];
              
              select4bind.id = item.name + '_sufix';
              bindData.map(function(item){
                  var op = document.createElement('option');

                  op.value = item;
                  op.innerText = item;
                  select4bind.appendChild(op);
              });

              labeldom.classList.add('bind-label');
              select4bind.classList.add('bind-sufix');
              labeldom.innerText = labelTitle;
              wrapDiv.appendChild(labeldom);
              wrapDiv.appendChild(select4bind);
           }

           setTimeout(function() {
             var domFn = $.domRenderDefaults[item.type || 'text'];
             $(input)[domFn]('setValue', '')[domFn]('setValue', item.value).next().hide();
           }, 0);

           var itemBt = eval('(' + item.button + ')');
           var btn = document.createElement('span'),
             btCls = {
               add: 'el-icon-plus',
               edit: 'formt-bt-edit'
             };
           if (btCls[itemBt.cls]) btn.classList.add(btCls[itemBt.cls]);
           if (itemBt.click && (typeof itemBt.click == 'function')) {
             // 绑定事件
             btn.addEventListener('click', function() {
               itemBt.click(item);
             });
           }
           btn.classList.add('form-bt');
           btn.classList.add('el-icon');
           wrapDiv.appendChild(btn);
         } else {

         }
       } else { // 列表域
         var options = 'width:"100%",',
           columns = 'columns:[[';
         var tb = document.createElement('table'),
           bar = document.createElement('div');
         div.classList.add('form-single');

         /*---------------columns start---------------*/
         if (item.operate && item.readonly != 1) {
             columns += '{field:"opts", title:"' + Render.status.operation +
               '", _tbId:"' + item.name + '",_idField:"' + item.idField +
               '", width:80, formatter: _tbOpFormatter,fixed:true}';
          }
         
         item.list.map(function(row, index) { // 生成columns
           if (columns != 'columns:[[') columns += ',';
           
           var cwidth = $.renderDefaultOptions.cwidth,
             isHidden = row.hidden == 1 ? true : false;
           if (row.suffix) row.label += '(' + row.suffix + ')';
           if (row.cwidth && row.cwidth != 'undefined' && row.cwidth != 'null') {
             cwidth = row.cwidth;
             columns += '{formatter: _columnFormatter,field:"' + row.name +
               '", title:"' + row.label + '", width:' + cwidth +
               ',fixed:true,hidden:' + isHidden + '}';
           } else {
             columns += '{formatter: _columnFormatter,field:"' + row.name +
               '", title:"' + row.label + '", width:' + cwidth + ',hidden:' +
               isHidden + '}';
           }

         });
         columns += ']]';
         options += columns;
         /*---------------columns end---------------*/
         wrapDiv.appendChild(bar);

         /*---------------title---------------*/
         var titleOp = document.createElement('span'),
           opAdd = document.createElement('span');
         titleOp.classList.add('group-operations');
         opAdd.classList.add('form-bt');
         opAdd.classList.add('el-icon');
         opAdd.classList.add('el-icon-plus');
         var btOpts = item.operations || ''
         if (btOpts) btOpts = eval('(' + btOpts + ')');
         // 扩展自定义按钮
         if(btOpts && btOpts.others) {
            btOpts.others.map(function(btItem){
              var btOther = $('<span class="el-icon"></span>'),
                  cwrap = $('<a class="linkbutton linkbutton_nowanna"></a>');
              
              btOther.addClass(btItem.cls);
              cwrap.css({
                display: 'flex',
                'align-items': 'center',
                padding: '2px 10px',
                'margin-top': '6px'
              });
              btOther.css({
                'margin-right': '10px'
              });

              if(btItem.click && typeof btItem.click == 'function') {
                cwrap.on('click', function() {
                  btItem.click.call(null, Render.originalGroups, btItem);
                });
              }

              cwrap.append(btOther).append(btItem.title);
              titleOp.appendChild(cwrap[0]);
            });
         }

         if (item.operations && btOpts.add == false) {

         } else titleOp.appendChild(opAdd);
         opAdd.addEventListener('click', function() {
           _renderProps(item.name, 0);
         });
         bar.classList.add('form-tb-title');
         bar.innerText = item.label;
         if (item.readonly != 1) bar.appendChild(titleOp);
         /*---------------datas start---------------*/
         if (item.url) options += ',url:"' + $.renderDefaultOptions.system + item.url +
           '"';
         /*---------------datas end---------------*/
         if (item.idField) options += ',idField: "' + item.idField + '"';
         if (item.pagination == true) {
           options += ',rownumbers: true, pagination: true';
         }
         if (item.onBeforeLoad) {
           options += ',onBeforeLoad: ' + item.onBeforeLoad;
         }
         if(item.max>=8) {
            options += ',rownumbers: true';
         }
         options += ',loadFilter: _tbLoadFilter,singleSelect: true,fitColumns: true,onLoadSuccess:function(data){$(this).datagrid("enableContextmenuAutoSize"); $(this).datagrid("resize");}';

         tb.setAttribute('data-options', options);
         $(tb).data('props', item);
         tb.setAttribute('id', item.name);
         tb.setAttribute('code', item.name);
         tb.classList.add('easyui-' + clsKeys[cKey]);
         wrapDiv.style.setProperty('min-height', '100px');
         wrapDiv.style.setProperty('flex-direction', 'column');
         wrapDiv.appendChild(tb);
       }

       $(div).append(wrapDiv);
       // 处理是否单行
       if (item.single == 1) div.classList.add('form-single');
       if (item.single == 2) div.classList.add('form-double');
       if (isSub == true && item.propHidden == 1) { /* 明细页隐藏的字段 */
         div.classList.add('form-hidden');
       } else if (isSub != true && item.type != 'list' && item.hidden == 1) { /* 主页隐藏的字段 */
         div.classList.add('form-hidden');
       }
       div.classList.add('form-item');
       if (item.validMsg) div.setAttribute('data-msg', item.validMsg);
     }
     /**
      * 初始化输入域属性
      **/
     function _initProps(input, props) { // 需要进一步详细实现
       if (props.suffix && props.label.indexOf('(' + props.suffix + ')')<0) props.label += '(' + props.suffix + ')';
       var type = props.type || 'text',
         options = '',
         opProps = { // 匹配组件属性 -- 转换
           max: 'max',
           min: 'min',
           url: 'url',
           height: 'height',
           precision: 'precision',
           prompt: 'prompt',
           prefix: 'prefix',
           //suffix: 'suffix',
           validType: 'validType',
           readonly: 'readonly',
           editable: 'editable'
         },
         eventstr = '',
         commonProps = ['name'],
         emths = [];
       if (type == 'select') { // combobox 特殊属性处理(data等)
    	 var comboData = props.data;
         if (props.data) options += 'data:' + comboData + ',';
         // 重写filter过滤规则，实现模糊匹配
         options += 'filter: function(q, row) { var opts = $(this).combobox("options"); return row[opts.textField].indexOf(q) >= 0; },';
       }
       props.height = $.renderDefaultOptions.inputHeight; // 输入域默认高度
       for (var prop in props) {
         var opProp = opProps[prop],
           propVal = props[prop];

         var boolTrans = ['readonly', 'editable', 'required'];
         if (boolTrans.includes(prop)) {
           propVal = propVal == 0 ? false : true;
         }
         if (typeof propVal != 'function') { // 避免对方法的迭代
           if (opProp) { // data-options里的属性
             if (propVal && typeof propVal != 'boolean') {
               if (['max', 'min'].includes(prop)) {
                 options += opProp + ':' + propVal + ','; // 非变量类型的值冒号后面加引号
               } else {
                 options += opProp + ':"' + propVal + '",'; // 非变量类型的值冒号后面加引号
               }
             } else if (typeof propVal == 'boolean') options += opProp + ':' + propVal + ',';
           } else if (commonProps.includes(prop)) { // 常规dom属性
             input.setAttribute(prop, propVal);
           } else if (prop == 'events' && propVal) {
             var eventsVal = eval('(' + propVal + ')');
             for (var key in eventsVal) { // JSON.stringify 无法解析包含变量的json字符，故手动拼接
               if (key == 'blur') emths.push(eventsVal[key]);
               else eventstr += key + ':' + eventsVal[key] + ','; // 变量类型的值冒号后面不加引号
             }
           } else if (prop == 'onChange' && propVal) {
             $(input).data('onchange', propVal);
           }
         } else {
           $(input).data('onchange', propVal);
         }
       };
       options += 'onChange: function(newV,oldV){ _onchageCallBack(newV,oldV,this); },';
       eventstr += 'blur:function(e){ _blurCallback(e);  }'; // blur事件特别处理，涉及到校验
       options += 'events: {' + eventstr + '},';
       options += 'value:"' + (props.value || props.dftValue || '') + '"';
       if (props.width) {
         options += ',width:"' + props.width + '"';
       } else if ($.renderDefaultOptions.inputWidth) {
         options += ',width:"' + $.renderDefaultOptions.inputWidth + '"';
       }
       if(type == 'multiple') {
         options += ',multiple: true';
       }
       input.setAttribute('data-options', options);
       $(input).data('props', props);
       $(input).data('fns', emths); // 容器存储blur事件(此blur是为了个性化校验)，方便_blurCallback集中执行

       input.setAttribute('id', props.name); // id属性和name相同
       input.setAttribute('code', props.name);
       input.setAttribute('oldValue', props.value || ''); // oldValue属性和value相同
     }
   }
   /* 级联处理 */
   function _parseCascade(value, props, isFirst) {
     if (props.cascade) {
       var casList = eval('(' + props.cascade + ')');
       casList.map(function(item) {
         /* show: 要显示的项、hide：要隐藏的项、filter：过滤后的项 */
         if (value == item.value) {
           if (item.show) {
             item.show.map(function(id) {
               try {
            	 var pForm = $('#' + props.name).parents('.form-item');
                 if(!pForm.hasClass('form-hidden')) {
                	 $('#' + id).parents('.form-item').removeClass('form-hidden');
                 }
               } catch (e) {
                 console.log(e);
               }
             });
           }
           if (item.hide) {
             item.hide.map(function(id) {
               try {
                 $('#' + id).parents('.form-item').addClass('form-hidden');
               } catch (e) {
                 console.log(e);
               }
             });
           }
           if (item.filter) {
             for (var nKey in item.filter) {
               var nVal = item.filter[nKey];
               if (typeof nVal != 'function') {
              	 if(isFirst == true) {
              		var opts = $('#' + nKey).combobox('options');

                  setTimeout(function(){
                    try {
                      $('#' + nKey).combobox('loadData', nVal);
                    } catch (e) {
                      console.log(e);
                    }
                  },0);

              		if(opts != val) return;
              	 }
                 try {
                   $('#' + nKey).combobox('loadData', nVal);

                   var val = $('#' + nKey).data('props').value,
	               	     values = nVal.map(function(item){ return item.value;});
                   if(values.includes(val)) $('#' + nKey).combobox('setValue', val);
                   else $('#' + nKey).combobox('setValue', nVal[0].value);
                 } catch (e) {
                   console.log(e);
                 }
               }
             }
           }
         }
       });
     }
   }
   /* change 回调 */
   function _onchageCallBack(newV, oldV, ctn) {
     var fn = $(ctn).data('onchange'),
       props = $(ctn).data('props'),
       fnName = $.domRenderDefaults[props.type],
       origVal = $(ctn)[fnName]('options').originalValue;
     if (newV !== '' && newV != origVal) {
       $(ctn).next('span').addClass('form-modified');
     } else {
       $(ctn).next('span').removeClass('form-modified');
     }
     _parseCascade(newV, props);
     if (fn) {
       if (typeof fn == 'function') fn(newV, oldV, props);
       else eval(fn)(newV, oldV, props);
     }
     if (props.type == 'select') {
       validateDom(props);
     }
   }
   /* blur 回调 */
   function _blurCallback(e) {
     var opts = $(e.target).parent().prev().data('option');
     validateDom(opts);
     try {
       $('#' + opts.name).data('fns').map(function(fun) {
         var result = fun(e, opts);
         // 自定义失去焦点接口有返回值说明校验不通过
         var ctn = $('#' + opts.name);
         if (result && typeof result == 'string' && !ctn.parents('.form-item').hasClass('form-hidden')) {
           ctn.parents('.form-item').addClass('invalid');
           ctn.parents('.form-item').attr('data-msg', result);
           Render.validResult.valid = false;
           Render.validResult.propsValid = false;
         }
       });
     } catch (e) {
       console.trace(e);
     }
   }
   /* 表格列展示转义 -- combobox组件类型 */
   function _columnFormatter(value, row, index) {
     var props = this,
       tb = $('table[data-options*="' + props.field + '"]'),
       pList = tb.data('props').list;
     $.each(pList, function(idx, item) {
       if (item.type == 'select' && item.data && props.field == item.name) {
         var cDatas = JSON.parse(item.data);
         $.each(cDatas, function(n, m) {
           if (m.value == value) value = m.text;
         });
       }
     });
     return value;
   }
   /* table 加载前处理 */
   function _tbLoadFilter(data) {
     var tb = $(this),
       opts = tb.data('props');
     if (isNaN(data.total)) {
       data = data.map(function(row) {
         if (opts.operate == true || opts.operate == 'true') {
           row._edit = true;
           row._remove = true;
         }
         return row;
       });
       data = {
         total: data.length,
         rows: data
       };
     } else { /* 操作按钮项 */
       data.rows = data.rows.map(function(row) {
         if (opts.operate == true || opts.operate == 'true') {
           row._edit = true;
           row._remove = true;
         }
         return row;
       });
     }
     if (opts.loadFilter) { // 过滤接口方法
       if ((typeof opts.loadFilter == 'function')) data = opts.loadFilter(data);
       else data = eval(opts.loadFilter).call(null, data);
     }
     /* hide add */
     if (opts.max) {
       if (data.rows.length >= opts.max) tb.parents('.form-item-wrap').find('.form-tb-title > .group-operations .el-icon-plus').hide();
     }
     if (opts.min) {
       if (data.rows.length <= opts.min) {
         data.rows = data.rows.map(function(row) {
           row._remove = false;
           return row;
         });
       }
     }
     try{
      var changeFn = opts.onChange;
      if(changeFn) {
        if (typeof changeFn == 'function') changeFn(data.rows, '', opts);
        else eval(changeFn)(data.rows, '', opts);
      }
      $(window).resize();
     }catch(e){}

     return data;
   }
   /* 初始化操作按钮项 */
   function _tbOpFormatter(value, row, index) {
     var opstr = '';
     if (row._edit == true) {
       opstr +=
         '<span class="form-bt el-icon el-icon-operation-edit" onclick="_renderProps(&quot;' +
         this._tbId +
         '&quot;,1,&quot;' + row[this._idField] +
         '&quot;)"></span>';
     }
     if (row._remove) {
       opstr +=
         '<span class="form-bt el-icon el-icon-operation-delete" onclick="_renderProps(&quot;' +
         this._tbId + '&quot;,2,&quot;' + row[this._idField] +
         '&quot;)"></span>';
     }
     return opstr;
   }
   /* table行数据明细层 */
   function _renderProps(tbId, flag, code) {

     var opTypes = {
       0: 'add',
       1: 'edit',
       2: 'remove'
     };
     var tb = $('#' + tbId),
       opts = tb.datagrid('options'),
       tbProps = tb.data('props'),
       idField = opts.idField,
       rows = tb.datagrid('getRows'),
       row = code ? rows.filter(function(row) {
         return row[idField] == code;
       })[0] : {},
       idFieldVal = row[idField] || ''; // 编辑行的主键值

     /* 记录index当前最大值 -- 以后可能会改掉这个逻辑 */
     var tbIndex = tb.data('index') || 1;
     rows.map(function(tRow) {
       var rIdx = tRow[idField] || 1;
       tbIndex = Math.max(tbIndex, rIdx);
     });
     tb.data('index', tbIndex);
     if (opTypes[flag] == 'add') {
       row[idField] = tbIndex * 1 + 1;
       tbProps.list.map(function(item){
          if(item.dftValue) row[item.name] = item.dftValue;
       })
     }

     if (opTypes[flag] == 'remove') {
       if(tbProps.events) {
          var eventsObj = eval('(' + tbProps.events + ')'),
              bool = eventsObj.blur(row, tb, tbProps);

          if(bool === false) {
            return;
          }
       }

       $.messager.confirm(Render.status.tips, Render.status.remove, function(r) {
         if (r) {
           var dJson = {};
           if (idField) dJson[idField] = idFieldVal;
           if (opTypes[flag]) dJson.operateType = opTypes[flag]; // 标记操作类型
           toTbDataQueue(tb, dJson, idField);
           var rIdx = tb.datagrid('getRowIndex', idFieldVal);
           tb.datagrid('deleteRow', rIdx);

           /* 最大最小数据量控制按钮显示逻辑(add、remove) */
           var remainRows = tb.datagrid('getRows');
           if (tbProps.max) {
             if (remainRows.length < tbProps.max) tb.parents('.form-item-wrap')
               .find('.form-tb-title>.group-operations .el-icon-plus').show();
           }
           if (tbProps.min) {
             if (remainRows.length <= tbProps.min) {
               remainRows.map(function(rowItem, idx) {
                 rowItem._remove = false;
                 tb.datagrid('updateRow', {
                   index: idx,
                   row: rowItem
                 });
               });
             }
           }
           
           try{
            var changeFn = tbProps.onChange;
            if(changeFn) {
                if (typeof changeFn == 'function') changeFn(remainRows, '', tbProps);
                else eval(changeFn)(remainRows, '', tbProps);
            }
           }catch(e) {}
         }
       }).addClass("seriousConfirm");
       return;
     }

     var oprts = tbProps.operations;
     if (oprts) oprts = eval('(' + oprts + ')');

     /* 如果有自定义的add、edit则不执行默认行为 */
     if (oprts && oprts.add != false && opTypes[flag] == 'add') {
       if (typeof oprts.add == 'function') {
         oprts.add.call(null, tbProps); //异常、校验、显示、数据行数等控制
       } else eval(oprts.add).call(null, tbProps);
       return;
     } else if (oprts && oprts.edit != false && opTypes[flag] == 'edit') {
       if (typeof oprts.add == 'function') {
         oprts.edit.call(null, tbProps, row); //异常、校验、显示、数据行数等控制
       } else eval(oprts.edit).call(null, tbProps, row);
       return;
     }

     var list = tbProps.list;
     list = list.map(function(item) {
       item.value = row[item.name] || '';
       var nItem = $.extend({},item);
       if(opTypes[flag] == 'add') {
    	   nItem.readonly = '0';
       }
       return nItem;
     });

     Render.pRender(list, document.querySelector('.form-ctn'), tbProps.label);
     $('.form-ctn .form-props .bt-group>a:first').off('click').on('click',
       function(event) {
         var pfm = $('.form-ctn .form-props form'),
           mJson = _getModifyData(pfm, opTypes[flag]),
           index = tb.datagrid('getRowIndex', idFieldVal);

         if (isEmptyJson(mJson)) { // 未填写数据或改动
           $.messager.alert('tips', 'No modifications');
           return;
         }
         if (!Render.validProps()) return; // 校验不通过

         if (opTypes[flag] == 'add') {
           var fDatas = pfm.serializeJSON();
           if (tb.data('props').operate == true || tb.data('props').operate == 'true') {
             fDatas._edit = true;
             fDatas._remove = true;
           }
           idFieldVal = fDatas[idField];
           tb.datagrid('appendRow', fDatas);
           /* 处理最大最小数据按钮控制逻辑(add、remove) */
           var hasRows = tb.datagrid('getRows');
           if (hasRows.length >= tbProps.max) {
             tb.parents('.form-item-wrap').find('.form-tb-title>.group-operations .el-icon-plus').hide();
           }

           if (hasRows.length >= tbProps.min) {
             hasRows.map(function(rowItem, idx) {
               if(idx == 0 && rowItem._remove === false) {

               }else {
                  rowItem._remove = true;
               }
               tb.datagrid('updateRow', { index: idx, row: rowItem});
             })
           }

           try{
            var changeFn = tbProps.onChange;
            if(changeFn) {
                if (typeof changeFn == 'function') changeFn(hasRows, '', tbProps);
                else eval(changeFn)(hasRows, '', tbProps);
            }
           }catch(e){}
         } else if (opTypes[flag] == 'edit') {
           tb.datagrid('updateRow', { index: index, row: mJson});
         }

         if (idField) mJson[idField] = idFieldVal;
         if (opTypes[flag]) mJson.operateType = opTypes[flag]; // 标记操作类型
         toTbDataQueue(tb, mJson, idField);
         openPropsPanel(false); // 保存成功后关闭属性修改层
       });
   }

   function isEmptyJson(json) {
     var bool = true,
       empty = {};
     for (var key in json) {
       var val = json[key];
       if ((typeof val != 'function') && val != empty[key]) bool = false;
     }
     return bool;
   }
   /* 将table的所有数据操作压入Render.tableCollector收集器中 */
   function toTbDataQueue(tb, row, idField) {
     var nameKey = tb.prop('id');
     if (!isArray(Render.tableCollector[nameKey])) Render.tableCollector[nameKey] = [];

     var exist = Render.tableCollector[nameKey].filter(function(item) {
       return item[idField] == row[idField];
     });
     var result = {
       is: true,
       msg: ''
     };
     /* 数据净化 */
     if (exist.length > 0) {
       if (exist[0].operateType == 'add') { // 历史记录是新增类型的
         if (row.operateType == 'add') {
           result.is = false;
           result.msg = 'exist';
         }
         if (row.operateType == 'edit') {
           row.operateType = 'add';
           Render.tableCollector[nameKey] = Render.tableCollector[nameKey].filter(
             function(item) { // 过滤掉历史操作
            	 return item[idField] != row[idField];
             });
         }
         if (row.operateType == 'remove') {
           result.is = false;
           Render.tableCollector[nameKey] = Render.tableCollector[nameKey].filter(
             function(item) {
            	 return item[idField] != row[idField];
             });
         }
       } else if (exist[0].operateType == 'edit') { // 历史记录是修改类型的
         if (row.operateType == 'add') {
           result.is = false;
           result.msg = 'exist';
         }
         if (row.operateType == 'edit') {
           Render.tableCollector[nameKey] = Render.tableCollector[nameKey].filter(
             function(item) {
            	 return item[idField] != row[idField];
             });
           row = $.extend({}, exist[0], row);
         }
         if (row.operateType == 'remove') {
           Render.tableCollector[nameKey] = Render.tableCollector[nameKey].filter(
             function(item) {
            	 return item[idField] != row[idField];
             });
         }
       } else if (exist[0].operateType == 'remove') { // 历史记录是删除类型的
         if (row.operateType == 'add') {
           Render.tableCollector[nameKey] = Render.tableCollector[nameKey].filter(
             function(item) {
            	 return item[idField] != row[idField];
             });
         }
         row.operateType = 'edit';
       }
     }
     if (result.is) Render.tableCollector[nameKey].push(row);
     else {
       if (result.msg) $.messager.show({
         title: 'info',
         msg: result.msg
       });
     }
   }
   /* 获取表单变动的数据 */
   function _getModifyData(form, type) {
     var params = $(form).serializeJSON(),
         mJson = {};
     for (var key in params) {
       var oldVal = $('#' + key, form).attr('oldvalue');

       if(params[key] != oldVal) mJson[key] = params[key];

       if(type == 'add' && params[key]) {
          mJson[key] = params[key];
       }
     }
     return mJson;
   }
   /* 获取整体表单变动数据 */
   function _getFormDatas(fm) {
     //Render.validResult.reboot = false;
     var form = $(fm),
       params = form.serializeJSON(),
       moJson = {};
     for (var key in params) {
       var inputItem = $('#' + key, form),
         props = inputItem.data('props'),
         fnName = $.domRenderDefaults[props.type],
         oldVal = inputItem.attr('oldValue');
       if (fnName == 'datebox') {
         oldVal = inputItem[fnName]('options').originalValue;
       }
       if (params[key] != oldVal && key.indexOf('_show') < 0) {
         moJson[key] = params[key];
         if (props.reboot == 1) Render.validResult.reboot = true;
       }
     }
     for (var tbName in Render.tableCollector) {
       moJson[tbName] = Render.tableCollector[tbName];
     }
     return moJson;
   }
   /* 获取嵌入的jsp的变动数据 */
   function _getJSPFormDatas(fm) {
     var params = {};
     $('[class*=easyui-]', fm).each(function(n, item) {
       var clses = item.classList;
       Array.from(clses).map(function(cls) {
         if (cls.includes('easyui-')) {
           var fnName = cls.replace('easyui-', ''),
             options = $(item)[fnName]('options'),
             val = $(item)[fnName]('getValue');
           if (val != options.originalValue) {
             params[$(item).attr('textboxname')] = val;
           }
         }
       });
     });
     return params;
   }
   /* jsp嵌入数据保存至table */
   function addJspDatasToTable(fm, tb, type) {
     var opts = $(tb).datagrid('options'),
       idField = opts.idField;

     if (!['add', 'edit', 'remove'].includes(type)) return;
     if (type == 'remove') {
       var sRow = $(tb).datagrid('getSelected'),
         rIdx = $(tb).datagrid('getRowIndex', sRow[idField]);
       sRow.operateType = type;

       $(tb).datagrid('deleteRow', rIdx);
       toTbDataQueue(tb, sRow, idField);
       return;
     }

     var row = Render.getJspDatas(fm),
       idFieldVal,
       tbProps = $(tb).data('props'),
       propIdField = idField;
     var namesMap = {},
       tbRow = {};
     /* 通过propName 实现对 name出现多值的问题  */
     tbProps.list.map(function(item) {
       var pkey = item.propName || item.name;
       namesMap[pkey] = item.name;
       if (item.name == idField) {
         idFieldVal = row[pkey];
         propIdField = pkey;
       }
     });
     for (var key in namesMap) {
       if (typeof key != 'function') {
         tbRow[namesMap[key]] = row[key];
       }
     }

     tbRow.operateType = type;
     /* 更新数据到表格 */
     var upRow = {};
     for (var key in namesMap) {
       if (typeof key != 'function') {
         if (row[key]) upRow[namesMap[key]] = row[key];
       }
     }
     if (type == 'add') {
       upRow._edit = true;
       upRow._remove = true;
       upRow[idField] = $(fm).serializeJSON()[propIdField];
       $(tb).datagrid('appendRow', upRow);
     } else if (type == 'edit') {
       upRow[idField] = idFieldVal;
       var stRow = $(tb).datagrid('getSelected'),
         index = $(tb).datagrid('getRowIndex', stRow[idField]);
       tbRow[idField] = stRow[idField];
       $(tb).datagrid('updateRow', {
         index: index,
         row: upRow
       });
     }

     toTbDataQueue(tb, tbRow, idField);
     openPropsPanel(false);
   }
   /**
    * 统一的校验方法（定制化强的校验由blur事件接口处理）
    * @param opts: 包含name、value、type等信息（其中name和id相同，方便检索）
    **/
   function validateDom(opts) {
     var msgs = Render.messages(opts);
     // 处理校验结果提示
     var ctn = $('#' + opts.name),
       code = _validCode();

     if (msgs[code]) {
       ctn.parents('.form-item').addClass('invalid');
       if (code != 'reboot' && !ctn.parents('.form-item').hasClass('form-hidden')) {
         Render.validResult.valid = false; // 排除重启提示的
         Render.validResult.propsValid = false;
       }
       if(code == 'reboot') Render.validResult.reboot = true;
    	   
     } else ctn.parents('.form-item').removeClass('invalid');
     ctn.parents('.form-item').attr('data-msg', msgs[code]);
     /* 校验处理，返回对应msgs里的键（valid、range等） */
     function _validCode() {
       var clsKeys = $.domRenderDefaults,
         type = opts.type || 'text',
         fnName = clsKeys[type];
       var val = ctn[fnName]('getValue');
       return validater(val, opts);

       function validater(val, ops) {
         var bools = [],
           isValid = true,
           code = 'valid';
         /* 必填校验 */
         if (ops.required == true && val == '') {
           code = 'required';
         } else {
           code = _threeRule(val, ops);
           /* 双输入域 */
           if (ops.type == 'range') {
             $('[name="' + ops.name + '"]').each(function(index, item) {
               var iVal = $(item).val();
               if (iVal) {
                 code = _threeRule(iVal, ops);
               } else if (ops.required == true) code = 'required';
             });
           }
         }

         return code;

         function _threeRule(iVal, ops) {
           iVal = iVal + '' || '';
           var code = 'valid';
           if (ops.type == 'num' && isNaN(iVal) && iVal != ops.value) return 'range';
           /* 重启校验 */
           bools = [];
           var orArr = iVal.split(',').sort(),
             newArr = ((ops.value || '')+'').split(',').sort()
           if (ops.reboot == true && orArr.join('') != newArr.join('')) {
             if (ops.type == 'num' && iVal - ops.value == 0) {

             } else code = 'reboot';
           }
           if (iVal && iVal != ops.value) {
             /* 大小校验 */
             if (!isNaN(iVal)) {
               if (ops.max) bools.push(iVal - ops.max <= 0);
               if (ops.min) bools.push(iVal - ops.min >= 0);
               if (ops.type == 'num' && ops.precision == 0) { //校验整型
                 var regInt = /^-?\d+$/;
                 bools.push(regInt.test(iVal));
               }
             }
             bools.map(function(bool) {
               if (!bool) code = 'range';
             });
             /* 长度校验 */
             bools = [];

             if (ops.minlength) bools.push(iVal.length >= ops.minlength);
             if (ops.maxlength) bools.push(iVal.length <= ops.maxlength);
             bools.map(function(bool) {
               if (!bool) code = 'length';
             });
           }

           return code;
         }
       }
     }
   }
   $.domRenderDefaults = {
     text: 'textbox',
     num: 'textbox',
     date: 'datebox',
     select: 'combobox',
     range: 'textbox',
     list: 'datagrid',
     bind: 'textbox',
     multiple: 'combobox'
   };
   $.renderDefaultOptions = {
     cwidth: 100,
     inputHeight: 25,
     inputWidth: 300
   };
   var Render = {
     gRender: groupRender, // 整体页面渲染器
     dRender: domRender, // dom渲染器
     pRender: propsRender, // 明细页渲染器
     tableCollector: {}, // 记录明细变动的集合
     getFormDatas: _getFormDatas, // 获取整体表单变动结果
     getJspDatas: _getJSPFormDatas,
     submit: function() {}, // 保存方法
     reset: function() {
       if (Render.originalCtner && Render.originalGroups) {
         $.messager.confirm(Render.status.tips, Render.status.confirm,
           function(r) {
             if (r) {
               $(Render.originalCtner).html('').addClass('loading');
               Render.gRender(Render.originalGroups, Render.originalCtner);
               setTimeout(function() {
                 $('.group-title:not(:first) .title-text', Render.originalCtner)
                   .click();
                 setTimeout(function() {
                   $(Render.originalCtner).removeClass('loading');
                 }, 500);
               }, 0);
             }
           }).addClass("seriousConfirm");
       }
     }, // 重置方法
     originalGroups: [],
     originalCtner: '',
     submitTxt: 'ok', // 保存按钮文本
     resetTxt: 'revoke', // 重置按钮文本
     propOkTxt: 'ok',
     propCancelTxt: 'cancel',
     validResult: {
       valid: true, // 整体表单校验结果
       propsValid: true, //明细表单校验结果
       reboot: false
     },
     valid: function() {
       Render.validResult.valid = true;
       var inputs = $('.form-ctn:not(.form-hidden) :input');
       inputs.each(function(index, item) { // 先执行常规校验
         var iName = $(item).prop('name');
         if (iName) {
           var opts = $('#' + iName).data('option');
           validateDom(opts);
         }
       }); // 这样确保校验逻辑优先级
       inputs.each(function(index, item) { // 在执行自定义校验
         var iName = $(item).prop('name');
         if (iName) {} else $(item).blur();
       });
       return Render.validResult.valid;
     },
     validProps: function() {
       Render.validResult.propsValid = true;
       var inputs = $('.form-props :input');
       inputs.each(function(index, item) { // 先执行常规校验
         var iName = $(item).prop('name');
         if (iName) {
           var opts = $('#' + iName).data('option');
           validateDom(opts);
         }
       }); // 这样确保校验逻辑优先级
       inputs.each(function(index, item) { // 在执行自定义校验
         var iName = $(item).prop('name');
         if (iName) {} else $(item).blur();
       });
       return Render.validResult.propsValid;
     },
     messages: function(opts) {
       var value = opts.value || '';
       if (opts.type == 'select') {
         var datas = $('#' + opts.name).combobox('getData');
         if (datas) {
           datas.map(function(row) {
             if (row.value == opts.value) value = row.text;
           });
         }
       }
       var msges = {
         valid: '',
         required: 'Need to fill',
         range: 'Int, min value: ' + opts.min + ', max value: ' + opts.max,
         length: 'Enter characters,length ' +
           opts.minlength + '-' + opts.maxlength,
         reboot: 'Need Reboot,raw value:' + value
       }
       return msges;
     },
     status: {
       success: 'success',
       operation: 'operations',
       tips: 'tips',
       confirm: '',
       remove: 'Are you sure to remove ?'
     }
   }
   /******************** domRender end ********************/
   /* ip与子网掩码  */
   function PoolIPCalculate(VUeAddr, Vnetmask){
		if(VUeAddr == '' && Vnetmask == '') return;

		var LowIPRange = "";
		var HighIPRange = "";

		if (VUeAddr == '')
		{
			VUeAddr = document.getElementById('LTE_LGW_START_UE_ADDR').value;
		}

		if (Vnetmask == '')
		{
			Vnetmask = getSelectedValue("LTE_LGW_NET_MASK");
			//alert("Vnetmask="+Vnetmask);
		}

		LowIPRange = getLowAddr(VUeAddr, Vnetmask);
		HighIPRange = getHighAddr(VUeAddr, Vnetmask);

		var Nrange = document.getElementById("netmask_range");
		Nrange.innerHTML = LowIPRange + ' - ' + HighIPRange;
	}

	function getLowAddr(ip, netMask){
	    var lowAddr = "";
	    var ipArray = new Array();
	    var netMaskArray = new Array();

	    if (4 != ip.split(".").length || netMask == "")
	    {
	        return "";
	    }
	    for (var i = 0; i < 4; i++)
	    {
	        ipArray[i] = ip.split(".")[i];
	        netMaskArray[i] = netMask.split(".")[i];
	        if (ipArray[i] > 255 || ipArray[i] < 0 || netMaskArray[i] > 255
	                && netMaskArray[i] < 0)
	        {
	            return "";
	        }
	        ipArray[i] = ipArray[i] & netMaskArray[i];
	    }

	    for (var i = 0; i < 4; i++)
	    {
	        if(i == 3)
			{
	            ipArray[i] = ipArray[i] + 1;
	        }
	        if (lowAddr == "")
			{
	            lowAddr +=ipArray[i];
	        } else{
	            lowAddr += "." + ipArray[i];
	        }
	    }
	    return lowAddr;
	}
	function getHighAddr(ip,netMask){
	    var lowAddr = getLowAddr(ip,netMask);
	    var hostNumber = getHostNumber(netMask);
	    if(lowAddr == "" || hostNumber == 0)
		{
	        return "";
	    }

	    var lowAddrArray = new Array();
	    for(var i = 0; i < 4; i++)
		{
	        lowAddrArray[i] = lowAddr.split(".")[i];
	        if(i == 3)
			{
	            lowAddrArray[i] = Number(lowAddrArray[i] - 1);
	        }
	    }
	    lowAddrArray[3] = lowAddrArray[3] + Number(hostNumber - 1);
	    //alert(lowAddrArray[3]);
	    if(lowAddrArray[3] > 255)
		{
	        var k = parseInt(lowAddrArray[3] / 256);
	        //alert(k);
	        lowAddrArray[3] = lowAddrArray[3] % 256;
	        //alert(lowAddrArray[3]);
	        lowAddrArray[2] = Number(lowAddrArray[2]) + Number(k);
	        //alert(lowAddrArray[2]);
	        if(lowAddrArray[2] > 255)
			{
	            k = parseInt(lowAddrArray[2] / 256);
	            lowAddrArray[2] = lowAddrArray[2] % 256;
	            lowAddrArray[1] = Number(lowAddrArray[1]) + Number(k);
	            if(lowAddrArray[1] > 255)
				{
	                k = parseInt(lowAddrArray[1] / 256);
	                lowAddrArray[1] = lowAddrArray[1] % 256;
	                lowAddrArray[0] = Number(lowAddrArray[0]) + Number(k);
	            }
	        }
	    }

	    var highAddr = "";
	    for(var i = 0; i < 4; i++)
		{
	        if(i == 3)
			{
				lowAddrArray[i] = lowAddrArray[i] - 1;
	        }
	        if(highAddr == "")
			{
	            highAddr = lowAddrArray[i];

	        }else{
	            highAddr += "." + lowAddrArray[i];
	        }
	    }

	    return highAddr;
	}
	function getHostNumber(netMask){
	    var hostNumber = 0;
	    var netMaskArray = new Array();
	    for(var i = 0; i < 4; i++)
		{
	        netMaskArray[i] = netMask.split(".")[i];
	        if(netMaskArray[i] < 255)
			{
	            hostNumber = Math.pow(256,3-i) * (256 - netMaskArray[i]);
	            break;
	        }
	    }

	    return hostNumber;
	}
	/* ip与子网掩码  */
  //防止按钮重复点击
   var disableButtonTimer;
   function forbiddenButton(){
	   var pNode = $(this).parent();
	   pNode.addClass('button-disabled');
	   pNode[0].addEventListener('click',function(e){
		   if($(this).hasClass('button-disabled')) e.stopPropagation();
		   _clearTimer();
	   },true);
	   _clearTimer();
	   
	   // 校验未通过的输入域滚动到可视窗口
	   setTimeout(locateErrors,0);
		
	   function _clearTimer(){
		   if(disableButtonTimer) clearTimeout(disableButtonTimer);
		   disableButtonTimer = setTimeout(function(){
			   pNode.removeClass('button-disabled');
		   },1000);
	   }
   }
   function locateErrors() {
	   var errorSelectors = [
		   	'.is-error',
		   	'.errorBorder',
		   	'.err_border',
		   	'[style*="color:red"]',
		   	'[style*="color: red"]',
		   	'[style*="border-color:red"]',
		   	'[style*="border-color: red"]'
		   ],
	   	   errorList = document.querySelectorAll(errorSelectors.join(','));
	   
	   if(errorList.length) { // scrollIntoView({behavior: 'smooth',block: 'start',inline: 'nearest'})
		   var  firstErr = errorList[0],
		   		inputs = firstErr.querySelectorAll('input,textarea');
		   
		   if(inputs.length) {
			   Array.from(inputs).map(function(input){
				   input.focus();
				   if(!isVisible(input)) {// 输入域在不可见区域时，无法触发滚动时，滚动容器
					  firstErr.scrollIntoView({behavior: 'smooth',block: 'end',inline: 'nearest'});
				   }
			   });
		   }else if(firstErr.tagName == 'DIV'){// 不含输入域时，滚动容器
			   firstErr.scrollIntoView({behavior: 'smooth',block: 'start',inline: 'nearest'});
		   }
		   
		   firstErr.focus();
	   }
   }
   /**
   * 初始化初始值
   * @param ctn: 初始化范围，默认document
   **/
   function initInputs(ctn){
     ctn = ctn || document;
     $(':input',$(ctn)).each(function(idx,domItem){
       domItem.setAttribute('value',domItem.value);
     });
   }
   /**
   * 检测值是否已变更
   * @param ctn: 初始化范围，默认document
   * @return boolean: true-有变动、 false-无变动
   **/
   function isValueChanged(ctn){
     ctn = ctn || document;
     var isChanged = false;
     $(':input',$(ctn)).each(function(idx,domItem){
       var originalValue;
       if(domItem.attributes.value) originalValue = domItem.attributes.value.value;
       if(domItem.value != originalValue) {
    	   isChanged = true;
       }
     });
     return isChanged;
   }
   function getChangedValue(ctn) {
     ctn = ctn || document;
     var form = {};
     $(':input',$(ctn)).each(function(idx,domItem){
       var originalValue;
       if(domItem.attributes.value) originalValue = domItem.attributes.value.value;
       if(domItem.value != originalValue) {
    	   form[domItem.name] = domItem.value;
       }
     });
     
     return form;
   }
   /**
    * 顶部消息提示方法
    * @param msg：提示消息
    * @param ctn：消息位于的容器（position是relative）
    * @param cls：自定义图标样式
    */
   function toast(msg,ctn,cls,auto){
       if (!ctn) ctn = document.body;
       if (msg == false) {
         $(event.target).parents('.top-tips').fadeOut(400,function(){
           $(this).remove();
           rePosition(ctn);
         });
         return;
       }
       var tipDom = _createTip();
       $(ctn).append(tipDom.css({top: rePosition(ctn)}));
       tipDom.fadeIn();
       
       /* 展示一定时间后自动清除 */
       if(auto == false) return;
       setTimeout(function(){
         tipDom.fadeOut(400,function(){
           $(this).remove();
           rePosition(ctn);
         });
       },4000);

       /* 创建容器 - 内部函数是可以共享外部函数的参数的 */
       function _createTip(){
         var tipCtn = $('<div class="top-tips">'+
                           '<div class="wrap">'+
                             '<div class="tip-info">'+
                               '<div class="tip-close" onclick="showTips(false)">x</div>'+
                               '<div class="tip-content">'+msg+'</div>'+
                             '</div>'+
                           '</div>'+
                         '</div>'),
             closeBt = tipCtn.find('.tip-close'),
             wrap = tipCtn.find('.wrap'),
             icon = $('<span class="tip-icon"> </span>');
         closeBt.on('click',function(){
        	 tipCtn.fadeOut(400,function(){
                 $(this).remove();
                 rePosition(ctn);
             });
         });
         if(cls) {
       	  tipCtn.addClass(cls);
             //icon.addClass(cls);
             //wrap.prepend(icon);
         }
         return tipCtn;
       }
       /* 重新定位 */
       function rePosition(ctn){
         var top = 5;
         $(ctn).find('.top-tips:visible').each(function(n,item){
           if((top+'px')!= $(item).css('top')) $(item).css({top:top});
           top += $(item).height() + 5;
         });
         return top;
       }
   }
 //设置操作列单元格样式 
   function setStyle(){
   	return 'position:relative';
   }
   //设备类型格式化
   function formatDiviceType(value,rowData,index){
		if(value == 0){
			value = "Type R(eNB)"
		}else if(value == 1){
			value = "Type Q(eNB)";
		}else if(value == 2){
			value = "OMC";
		}else if(value == 3){
			value = "EPC";
		}else if(value == 4){
			value = "eGW";
		}
		return value;
	}
   function validateInput(ele,message){
		var value = $(ele).val();
		var length = value.length;
		var maxLength = $(ele).attr("max_length");
		var paramType = $(ele).attr("paramType");
		var validateMsg = JSON.parse(message);
		if (length > maxLength) {
			var msg = validateMsg.ChangDuChaoChuFanWei + maxLength + validateMsg.ZiFu;
			$(ele).siblings(".prompt").text(msg).show();
		}else{
			if(paramType == "phone"){
				var reg = /^[0-9-+() ]*$/;
				if(value == "" || reg.test(value)){
					$(ele).siblings(".prompt").hide();
				}else{
					$(ele).siblings(".prompt").show().text(validateMsg.phoneMessage);
				}
			}else if(paramType == "IdNumber"){
				var reg = /^[a-zA-Z0-9-_ ]*$/;
				if(value == "" || reg.test(value)){
					$(ele).siblings(".prompt").hide();
				}else{
					$(ele).siblings(".prompt").show().text(validateMsg.ShenFenZhengGeShi);
				}
			}
		}
	}
   
Array.prototype.evaluate = function(str){
	this.map(function(item, idx){
		var reg = new RegExp('{\\s*\\$' +idx+ '\\s*}','g');
		str = str.replace(reg, item);
	});
	return str;
}
String.prototype.evaluate = function(map){
	var txt = this.toString();
	for(var key in map){
		if(map.hasOwnProperty(key)){
			var reg = new RegExp('{\\s*' +key+ '\\s*}','g');
			txt = txt.replace(reg, map[key]);
		}
	}
	return txt;
}
//regs 可以匹配的文件格式 -以逗号分隔的 字符串
//str 选择的内容 - 字符串
function fileFormatMatch(str,regs){
	var regsArr = regs.toLowerCase().split(",");
	var suffix = str.substring(str.lastIndexOf(".")+1).toLowerCase();
	if(regsArr.indexOf(suffix)>-1){
		return true;
	}else{
		return false;
	}
}
//axios参数格式化
function stringify(json){
	   var param = new URLSearchParams();
	   if(json){
		   for(var key in json){
			   if(json.hasOwnProperty(key)){
				   param.append(key,json[key])
			   }
		   }
	   }
	   return param;
}
//form表单是否发生改变
function isFormChanged(form){
	   var isChanged = false;
	   if(form.isReinited){
		   form.fields.map(function(field){
			   if(Array.isArray(field.fieldValue)){
				   var vList = field.fieldValue.map(function(item){return item});
				   var oList = (field.reinitialValue||[]).map(function(item){return item});
				   var val = JSON.stringify(vList.sort());
				   var orVal = JSON.stringify(oList.sort());
				   if(val != orVal) isChanged = true;
			   }else{
				   if(isNull(field.fieldValue) && isNull(field.reinitialValue)){
					   
				   }else if(field.fieldValue !== field.reinitialValue) isChanged = true;
			   }
		   })
	   }else{
		   form.fields.map(function(field){
			   if(Array.isArray(field.fieldValue)){
				   var vList = field.fieldValue.map(function(item){return item});
				   var oList = (field.initialValue||[]).map(function(item){return item});
				   var val = JSON.stringify(vList.sort()); 
				   var orVal = JSON.stringify(oList.sort());
				   if(val != orVal) isChanged = true;
			   }else{
				   if(isNull(field.fieldValue) && isNull(field.initialValue)){
					   
				   }else if(field.fieldValue !== field.initialValue) isChanged = true;
			   }
		   })
	   }
	   return isChanged;
	   function isNull(val){
		   if(val==undefined || val == null || val =="") return true;
		   else return false;
	   }
	   
}
function initForm(form, props){
	   form.isReinited = true;
	   form.fields.map(function(field){
       if(props) {
          if(props.includes(field.prop)) field.reinitialValue = field.fieldValue;
       }else {
          if(Array.isArray(field.fieldValue)) {
            field.reinitialValue = JSON.parse(JSON.stringify(field.fieldValue));
          }else {
		        field.reinitialValue = field.fieldValue;
          }
       }
	   });
}
/**
 * 检测表单域状态，如有值变动是否显示重启提示
 **/
function detectReboot(form, rebootMap) {
  var bool = false;

  if(form) {
    form.fields.map(function(field){
      if(form.isReinited){
        form.fields.map(function(field){
          if(Array.isArray(field.fieldValue)){
            var vList = field.fieldValue.map(function(item){return item});
            var oList = (field.reinitialValue||[]).map(function(item){return item});
            var val = JSON.stringify(vList.sort());
            var orVal = JSON.stringify(oList.sort());
            if(val != orVal) {
              if(rebootMap[field.prop]) {
                field.$el.classList.add('need-reboot');
                bool = true;
              }
            }else {
              field.$el.classList.remove('need-reboot');
            }
          }else{
            if(isNull(field.fieldValue) && isNull(field.reinitialValue)){
              field.$el.classList.remove('need-reboot');
            }else if(field.fieldValue !== field.reinitialValue) {
              if(rebootMap[field.prop]) {
                field.$el.classList.add('need-reboot');
                bool = true;
              }
            }else{
              field.$el.classList.remove('need-reboot');
            }
          }
        })
      }else{
        form.fields.map(function(field){
          if(Array.isArray(field.fieldValue)){
            var vList = field.fieldValue.map(function(item){return item});
            var oList = (field.initialValue||[]).map(function(item){return item});
            var val = JSON.stringify(vList.sort()); 
            var orVal = JSON.stringify(oList.sort());
            if(val != orVal) {
              if(rebootMap[field.prop]) {
                field.$el.classList.add('need-reboot');
                bool = true;
              }
            }else {
              field.$el.classList.remove('need-reboot');
            }
          }else{
            if(isNull(field.fieldValue) && isNull(field.initialValue)){
              field.$el.classList.remove('need-reboot');
            }else if(field.fieldValue !== field.initialValue) {
              if(rebootMap[field.prop]) {
                field.$el.classList.add('need-reboot');
                bool = true;
              }
            }else {
              field.$el.classList.remove('need-reboot');
            }
          }
        })
      }
    }); 
  }

  return bool;

  function isNull(val){
    if(val==undefined || val == null || val =="") return true;
    else return false;
  }
}

function showMsg(type,msg){
	$('.'+type).html(msg);
	var width = $('.'+type).width();
	$('.'+type).css("left",'50%');
	var left = parseFloat($('.'+type).css("left")) - width/2;
	$('.'+type).css("left",left+'px');
	$('.'+type).animate({top:'55px'},200,function(){
		setTimeout(function(){
			$('.'+type).animate({top:'-40px'})
			$('.'+type).html('');
		},5000)
	})
}
function closeLicenseTip(){
	$('.licenseTooltip').hide().css("top","-65px").show();
}
//IPV6地址判断
function isIPv6(str) {
    return /:/.test(str)&&str.match(/:/g).length<8&&/::/.test(str)?
    (str.match(/::/g).length==1&&/^::$|^(::)?([\da-f]{1,4}(:|::))*[\da-f]{1,4}(:|::)?$/i.test(str)):
    /^([\da-f]{1,4}:){7}[\da-f]{1,4}$/i.test(str);
}
//IPV4地址判断
function isIPv4(str) {
    var reg = /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-4]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/;
    return reg.test(str);
}
//校验是否有效IP (IPv4或IPv6)
function isValidIP(str) {
	return isIPv4(str) || isIPv6(str);
}
// 通过form方式下载文件
function exportByForm(url,param) {
	var body = document.querySelector('body'),
		form = document.createElement('form'),
		params = param || {};
	
	form.style.display = 'none';
	form.action = url;
	form.method = 'post';
	
	if(params) {
		params.token = omctoken;
		for(var key in params) {
			var input = document.createElement('input');
			input.value = params[key];
			input.setAttribute('name',key);
			form.appendChild(input);
		}
	}
	
	body.appendChild(form);
	form.submit();
	form.remove();
}

// 前端分页 -- 将datagrid的loadFilter属性设置为这个方法名即可
function partPurchasePagerFilter(data) {
    if (typeof data.length == 'number' && typeof data.splice == 'function') {
        data = {
            total : data.length,
            rows : data
        }
    }
    var dg = $(this);
    var opts = dg.datagrid('options');
    var pager = dg.datagrid('getPager');
    pager.pagination({
        onSelectPage : function(pageNum, pageSize) {
            opts.pageNumber = pageNum||1;
            opts.pageSize = pageSize;
            pager.pagination('refresh', {
                pageNumber : pageNum,
                pageSize : pageSize
            });
            dg.datagrid('loadData', data);
        },
        displayMsg: 'Total {total}',
        layout: ['list','sep','prev','sep','manual','sep','next','sep','refresh']
    });
    if (!data.originalRows) {
        data.originalRows = (data.rows);
    }
    var start = (opts.pageNumber - 1) * parseInt(opts.pageSize);
    var end = start + parseInt(opts.pageSize);
    
    if(opts.queryParams.searchText && opts.queryParams.likeFields) {// 实现前端查询过滤
        var sTxt = (opts.queryParams.searchText || '').trim(),
            fields = opts.queryParams.likeFields;
        var nRows = data.originalRows.filter(function(row){
                        var isMatch = false;
                        fields.split(',').map(function(field) {
                             if (sTxt && row[field] && row[field].indexOf(sTxt) < 0) {
                                  
                             } else if(row[field]){
                                isMatch = true
                             }
                        });
                        return isMatch;
                    });
        data.total = nRows.length;
        data.rows = (nRows.slice(start, end));
    }else {
        data.rows = (data.originalRows.slice(start, end));
        data.total = data.originalRows.length;
    }
    
    return data;
}

/**
* 前端方式查询表格
* @param tb (object): 表格对象
* @param fields(string): 匹配的字段属性（多个以逗号分隔）
* @param text(string): 检索的文本
* @eg: searchFun($('#userTable'), 'name,sex', '赵公子');
**/
function searchFun(tb,fields,text) {
    // 传递查询参数
    $.extend(tb.datagrid('options').queryParams,{searchText: text,likeFields: fields});
    // 触发表格数据前端刷新
    tb.parents('.datagrid-wrap').find('.pagination-load').click();
}
/* 防抖函数 */
function debounce(fn,delay=200){
	let timer = null;
	return function(){
    var _self = this,
        _arg = arguments;
        
		if(timer) clearTimeout(timer);
		timer = setTimeout(function(){
			fn.apply(_self,_arg);
			timer = null;
		},delay);
	}
} 
/* 属性变动检测器 */
var MutationObserver = window.MutationObserver || window.WebKitMutationObserver || window.MozMutationObserver;
var observer = new MutationObserver(debounce(function(mutations) {
	mutations.forEach(function(mutation) {
    if(mutation.attributeName == 'style') {
      var style = getComputedStyle(mutation.target),
          right = (style.right||'').replace('px',''),
          masks = document.querySelectorAll('.slidebarPanel-shadow');
      
      if(isElementInViewport(mutation.target) && style.display != 'none'){// 滑入
        if(masks.length == 0) {
          var node = document.createElement('div');
          node.classList.add('slidebarPanel-shadow');
          document.body.appendChild(node);
        }
      }else{// 滑出
        Array.from(masks).map(function(item){
          item.remove();
        })
      }
      if(mutation.oldValue){// 检测是否触发退出
          var res = mutation.oldValue.match(/right:\s*([-0-9\.]+)px/),
              right = (style.right||'0').replace('px','');
          if(res && right%10 == 0 ){
            if( res[1] - right>0) {
              $(mutation.target).slideUp()
            }else if(res[1] - right!=0){
              $(mutation.target).slideDown(function(){
            	  try{
                	  $('table.datagrid-f').datagrid('resize');
                	  window.dispatchEvent(new Event('resize'));
            	  }catch(e){}
              })
            }
          }
      }
    }
	});
}));
/* 初始化滑出层遮罩 */
function initSlideShadow(context) {
	var containers = ['.slidebarPanel']; // 要监测的容器队列、方便以后扩展
	
	containers.map(function(selector){
		var layers = (context||document).querySelectorAll(selector);
		
		Array.from(layers).map(function(layer){
			var isObserved = layer.getAttribute('observed');
			if(isObserved != 'yes') { // 防止重复监测
				observer.observe(layer,{
					attributes: true, // configure it to listen to attibute changes
					attributeOldValue: true,
					attributeFilter: ['style']
				});
				layer.setAttribute('observed','yes');
			}
		});
	})
}

function isElementInViewport(el) {
  var rect = el.getBoundingClientRect();
  return ( rect.right <= (window.innerWidth || document.documentElement.clientWidth));
}

function uploadWithProgress({url,form,progress,success}) {
	if(url && form) {
		var formObj = new FormData(form),
			xhr = new XMLHttpRequest();

		formObj.append('token',omctoken);
		xhr.open('post',url);
		
		xhr.onreadystatechange = function() {
			// readystate为4表示请求已完成并就绪
			if(this.readyState == 4 && xhr.status == 200) {
				var data = JSON.parse(xhr.responseText)
				if(success) success(data);
			}
		}
		// 检测上传进度
		xhr.upload.onprogress = function(ev) {
			if(progress) progress(ev);
		}
		
		xhr.send(formObj);
	}
}
/**
* 判断元素是否可见
* @param el{dom}: dom元素
* @eg: isVisible(document.querySelector(cssSelector));
**/
function isVisible(el) {
    var loopable = true,
        visible = getComputedStyle(el).display != 'none' && getComputedStyle(el).visibility != 'hidden';
    
    while(loopable && visible) {
        el = el.parentNode;
        if(el && el != document.body) {
            visible = getComputedStyle(el).display != 'none' && getComputedStyle(el).visibility != 'hidden';
        }else {
            loopable = false;
        }
    }
    
    return visible;
}

/**
 * 初始化任务状态
 * 
 **/
function initTaskStatus(code, menus) {
  /**
   * 1 - 等待     2 - 进行中     3 - 暂停
   * 4 - 已结束   5 - 终止中     6 - 暂停中
   */
  var codes = {
        1: 'waitting',
        2: 'progress',
        3: 'suspend',
        4: 'finished',
        5: 'terminating',
        6: 'suspendding'
      },
      taskStatusMap = {
        '结果': 'result',
        '开始': 'start',
        '暂停': 'wait',
        '终止': 'end',
        '删除': 'del',
        '修改': 'mod',
        '信息': 'info',
        'Results': 'result',
        'Start': 'start',
        'Suspend': 'wait',
        'Terminate': 'end',
        'Delete': 'del',
        'Modify': 'mod',
        'Information': 'info'
      },
      status = codes[code],
      optsMap = {
        info: {show: true, disable: false, cls: 'el-icon el-icon-operation-info'},
        result: {show: true, disable: false, cls: 'el-icon el-icon-operation-result'},
        start: {show: true, disable: false, cls: 'el-icon el-icon-operation-start'},
        wait: {show: true, disable: false, cls: 'el-icon el-icon-operation-awaiting'},
        end: {show: true, disable: false, cls: 'el-icon el-icon-operation-terminate'},
        del: {show: true, disable: false, cls: 'el-icon el-icon-operation-delete'},
        mod: {show: false, disable: true, cls: 'el-icon el-icon-operation-edit'}
      };
  // 等待、暂停和暂停中状态处理
  if( ['waitting','suspend','suspendding'].includes(status) ) {
    Object.assign(optsMap,{
      wait: {show: false, disable: false, cls: 'el-icon el-icon-operation-awaiting'}
    });
    // waitting状态编辑按钮可用
    if(status == 'waitting') {
      optsMap['mod'].disable = false;
      optsMap['mod'].show = true;
      optsMap['info'].show = true;
    }
  }
  // 进行中状态处理
  if(status == 'progress') {
    Object.assign(optsMap,{
      start: {show: false, disable: false, cls: 'el-icon el-icon-operation-start'},
      del: {show: true, disable: true, cls: 'el-icon el-icon-operation-delete'}
    });
  }
  // 完成和终止中状态处理
  if( ['finished','terminating'].includes(status) ) {
    Object.assign(optsMap,{
      start: {show: true, disable: true, cls: 'el-icon el-icon-operation-start'},
      wait: {show: false, disable: false, cls: 'el-icon el-icon-operation-awaiting'},
      end: {show: true, disable: true, cls: 'el-icon el-icon-operation-terminate'},
    });
  }
  // 菜单状态更新
  menus.map(function(row){
    var key = row.label || row.text,
        item = optsMap[taskStatusMap[key]];
    if(item) {
      var rowCls = row.cls||'';
      Object.assign(row, {
        show: item.show,
        disable: item.disable,
        cls: rowCls +' '+ item.cls
      });
    }
  })
}

function loadHTML(ctn,params) {
	var url = params.url,
		method = params.method || 'get',
		cb = params.success,
		queryData = params.queryParams||{};
	
	if(!url) return;
		
	$.ajax({
        type: method,
        url: url,
        data: queryData,
        dataType: 'html',
        success: function(html) {
          ctn.innerHTML = html;
          
          var scripts = ctn.querySelectorAll('script');
          setTimeout(function() {
            Array.from(scripts).map(function(script) { /* 执行远程的脚本 */
              if (ctn.contains(script)) {
            	  try{
                	  ctn.removeChild(script);
            	  }catch(e){}
              }
              var newScript = document.createElement('script');
              newScript.type = 'text/javascript';
              newScript.innerHTML = script.innerHTML;
              ctn.appendChild(newScript);
            });
            if(cb && typeof cb == 'function') cb();
          }, 0);
          
          setTimeout(function() {
              try{
            	  $.parser.parse(ctn);
              }catch(e){}
          }, 0);
        },
        error: function() {
          if(cb && typeof cb == 'function') cb();
        }
	});
}
// 检测元素是否被遮蔽
function isOverlapped(element) {
	var document = element.ownerDocument,
	    rect = element.getBoundingClientRect(),
	    x = rect.x, 
	    y = rect.y, 
	    width = rect.width, 
	    height = rect.height;

	x |= 0;
	y |= 0;
	width |= 0;
	height |= 0;

	var elements = [
	    document.elementFromPoint(x+2, y+2),
	    document.elementFromPoint(x + width-2, y+2),
	    document.elementFromPoint(x+2, y + height-2),
	    document.elementFromPoint(x + width-2, y + height-2)
    ];
  
	return elements.filter((el)=> el !== null).some((el)=> {
	    return el !== element && !element.contains(el);
	});
}

function autoSetTitles(dg)  {
  var db = $(dg).data('datagrid'),
      ops = db.options,
      columns = ops.columns[0];

  columns.map(function(col){
    var context = $(dg).siblings("div.datagrid-view2");
          th = $('.datagrid-header-row td[field='+col.field+'] div.datagrid-cell',context);

      th.attr('title',th.text());
  });
}

/**
* 提示浮层 -- PopLayer
* 
* @event {Event}: 鼠标事件对象，用于自动定位
* @html {String}: 提示的内容
* @show {Boolean}: 主要用于关闭 -- false
**/
function showPopLayer({event,html='',show=true}) {
  _remove();

  if(show === false) {
    return;
  }

  var div = document.createElement('div'),
      wrap = document.createElement('div');

  div.classList.add('pop-layer');
  wrap.style.position = 'relative';
  div.appendChild(wrap);
  wrap.innerHTML = html;

  // create close span
  var closeSpan = document.createElement('span');
  closeSpan.style.position = 'absolute';
  closeSpan.style.zIndex = 100;
  closeSpan.style.padding = '0 5px';
  closeSpan.style.top = 0;
  closeSpan.style.right = 0;
  closeSpan.classList.add('panel_close');

  wrap.prepend(closeSpan);

  document.body.appendChild(div);
  _addEvent(div);

  var winHeight = document.body.clientHeight,
      winWidth = document.body.clientWidth,
      divRec = div.getBoundingClientRect(),
      divHeight = divRec.height,
      divWidth = divRec.width;
  // 设置 pop layer 位置 越界处理
  if(event.clientY + divHeight > winHeight) {
    div.style.top = winHeight - divHeight - 10;
  }else {
    div.style.top = event.clientY;
  }

  if(event.clientX + divWidth > winWidth) {
    div.style.left = winWidth - divWidth - 10;
  }else {
    div.style.left = event.clientX;
  }

  event.stopPropagation();

  // pop layer 清除
  function _remove() {
    document.body.removeEventListener('click',hidePopLayer);

    Array.from(document.querySelectorAll('.pop-layer')).map(function(item){
      item.remove();
    })
  }
  // 事件绑定
  function _addEvent(div) {
    closeSpan.addEventListener('click',function(evt){
      div.remove();
    });

    div.addEventListener('click',function(evt) {
      evt.stopPropagation();
    });

    document.body.addEventListener('click',hidePopLayer);
  }
}

function hidePopLayer() {
  showPopLayer({show:false});
}


//判断密码中是否至少包含字符，数字和密码中的两种，如果是返回0，不是返回1
function checkPasswdStrength(password) {
	var pattern_d_contain = /\d+/;      //包含数字
	var pattern_s_contain = /[A-Za-z_]+/ //包括字符
	var pattern_w = /^\w+$/;            //数字或者字符
	var pattern_r = /^\w+[\w\W]*\W+[\w\W]*$/    //以字母或者数字开头结尾的字符串
  var allReg = /^(?![A-Za-z0-9]+$)(?![a-z0-9_!@#$%^&*?]+$)(?![A-Za-z_!@#$%^&*?]+$)(?![A-Z0-9_!@#$%^&*?]+$)[a-zA-Z0-9_!@#$%^&*?]{6,20}$/; // 数字、大写字母、小写字母和特殊符号

  if(allReg.test(password)) {
    return "0";
  /*
  }else if (pattern_w.test(password) && pattern_d_contain.test(password) && pattern_s_contain.test(password)) {//密码只包含单词字符和数字，密码强度中
		return "0";
	} else if (pattern_r.test(password) && !pattern_d_contain.test(password)) {//密码只包含单词字符和特殊字符，密码强度中
		return "0";
	} else if (pattern_r.test(password) && !pattern_s_contain.test(password)) {//密码只包含数字和特殊字符，密码强度中
		return "0";
	} else if (pattern_r.test(password)) { //密码包含单词字符，数字和特殊字符，密码强度高
		return "0";
  */
	} else {
		return "1";
	}
}

/** 
* 数值大小比较（小于等于） -- 支持超过16位数
* @param min{string}: 要比较的数值
* @param max{string}: 目标数
**/
function isLessThan(min, max) {
    var max = (max + '').replace(/^[\s0]*/,''),
        min = (min + '').replace(/^[\s0]*/,''),
        bool = true,
        list = [];

    if(min.length > max.length || isNaN(min)) {
        bool = false;
    }

    if(min.length == max.length) {
        for(var i = 0; i < max.length; i++) {
            var isBig = max[i] - min[i] >= 0;

            list.push(max[i] - min[i] > 0);

            if(!isBig) {
                var some = list.filter(function(item){ return item == true;});

                if(some.length == 0) {
                    bool = false;
                    break;
                }
            }
        }
    }

    return bool;
}

function isMobile() {
  let match = window.matchMedia('(pointer:coarse)');

  return match && match.matches;
}

function addMobileStyle() {
  let style = document.createElement('style');

  style.innerHTML = `
    body {
      zoom: 2;
    }
  `;

  document.body.appendChild(style);
}

function checkMobileOnWindowResize() {
    if(isMobile()) {
      document.body.classList.add('body-zoom');
    }else {
      document.body.classList.remove('body-zoom');
    }
}
// 小站密码特殊校验规则
function checkPasswordSpecialRule(str) {
  let boolMap = {
          orderValid: true,   // 序列校验结果，针对：abc、456等
          yearValid: true,    // 年份校验结果，针对：1970-2050等
          repeatValid: true,  // 重复校验结果，针对：aa、AA等
          fourValid: true,    // 四组合校验结果
          threeValid: true    // 字符、数字、特殊符号
      },
      threeReg = /^(?=.*?[a-zA-Z])(?=.*?[0-9])(?=.*?[_!@#$%^&*?])[a-zA-Z0-9_!@#$%^&*?]{12,64}$/, //字符、数字、特殊符号
      fourReg = /^(?![A-Za-z0-9]+$)(?![a-z0-9_!@#$%^&*?]+$)(?![A-Za-z_!@#$%^&*?]+$)(?![A-Z0-9_!@#$%^&*?]+$)[a-zA-Z0-9_!@#$%^&*?]{12,64}$/, // 四种组合
      repeatReg = /(.)\1{1,}/g,
      allOrder = ['abcdefghijklmnopqrstuvwxyz','ABCDEFGHIJKLMNOPQRSTUVWXYZ','0123456789'];

  boolMap.fourValid = fourReg.test(str);
  boolMap.threeValid = threeReg.test(str);

  boolMap.repeatValid = (str.match(repeatReg)||'').length == 0;
  
  // 校验连续字符和数字: 如ab、12 
  str.split('').reduce((n, m)=>{ 
      let comb = n + m;
      let orderNum = 2;
      let orderStr = comb.slice(comb.length - orderNum);
      
      if(orderStr.length == orderNum && allOrder.some((m)=>{ return m.indexOf(orderStr) > -1;})) {
          boolMap.orderValid = false;
      }
      
      return n + m;
  })
  // 校验年份
  str.split('').reduce((n, m)=>{ 
      let range = [1970, 2050],
          comb = n + m,
          charLength = 4,
          orderStr = comb.slice(comb.length - charLength);
      
      if(orderStr.length == charLength && orderStr - range[0] >= 0 && orderStr - range[1] <= 0) {
          boolMap.yearValid = false;
      }
      
      return n + m;
  })

  return boolMap;
}
/**
  * 下载文件 - 带进度监控
  * @param url: 文件请求路径
  * @param params: 请求参数
  * @param filename: 保存的文件名
  * @param progress: 进度处理回调函数
  * @param success: 下载完成回调函数
  * eg: progressDownLoad({url:'http://loacalhost:8080/downLoad.action',filename:'file.rar',progress:function(evt){
  *        console.log(evt);
  *     }});
  **/
function progressDownLoad({url,filename,params,progress,success}){
    var xhr = new XMLHttpRequest();
    xhr.open("POST", url, true);
    //监听进度事件
    xhr.addEventListener("progress", function (evt) {
        if(progress) try{ progress.call(evt); }catch(e){}
    }, false);

    xhr.responseType = "blob";
    xhr.setRequestHeader("Content-Type","application/x-www-form-urlencoded; charset=UTF-8");
    xhr.onreadystatechange = function () {
        if (xhr.readyState === 4 && xhr.status === 200) {
            if (typeof window.chrome !== 'undefined') {
                // Chrome version
                var link = document.createElement('a');
                link.href = window.URL.createObjectURL(xhr.response);
                link.download = filename;
                link.click();
            } else if (typeof window.navigator.msSaveBlob !== 'undefined') {
                // IE version
                var blob = new Blob([xhr.response], { type: 'application/force-download' });
                window.navigator.msSaveBlob(blob, filename);
            } else {
                // Firefox version
                var file = new File([xhr.response], filename, { type: 'application/force-download' });
                window.open(URL.createObjectURL(file));
            }
            if(success) try{ success.call(xhr); }catch(e){}
        }
    };
    // FormData
    //var formData = new FormData();
    var paramsStr = '';
    if(params) for (var key in params) paramsStr += '&'+key+'='+params[key];
    if(paramsStr) paramsStr = paramsStr.substring(1);

    xhr.send(paramsStr);
}
// 下载文件
function downLoadFileByAxios(url,params){
    axios.post(url, stringify(params),{
        responseType: 'blob'
    }).then(function(res){
        const contentDisposition = res.headers['content-disposition'] || res.headers['Content-Disposition'];
        let fileName = 'downloaded-file';
        if (contentDisposition && contentDisposition.indexOf('attachment') !== -1) {
            const matches = /filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/.exec(contentDisposition);
            if (matches != null && matches[1]) {
                fileName = matches[1].replace(/['"]/g, '');
            }
        }

        // download file
        const url = window.URL.createObjectURL(res.data);
        const link = document.createElement('a'); 
        link.href = url;
        link.setAttribute('download', fileName);
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        window.URL.revokeObjectURL(url);
    })
}

// 绘制扇面
function drawSignal() {
  // cell-1
  drawPie({
    selector: '.cell-sector-ctn .cell-1',
    radius: cellOptions.radius,        // 外半径
    minRadius: cellOptions.minRadius,  // 内半径
    angles: cellOptions.angles,        // 弧度
    direct: cellOptions.direct || 0.1,        // 方向
    translate: cellOptions.translate
  });
  /*
  // cell-2
  drawPie({
    selector: '.cell-sector-ctn .cell-2',
    radius: cellOptions.radius,        // 外半径
    minRadius: cellOptions.minRadius,  // 内半径
    angles: cellOptions.angles,        // 弧度
    direct: cellOptions.direct + 120,  // 方向
    translate: cellOptions.translate
  });
  // cell-3
  drawPie({
    selector: '.cell-sector-ctn .cell-3',
    radius: cellOptions.radius,        // 外半径
    minRadius: cellOptions.minRadius,  // 内半径
    angles: cellOptions.angles,        // 弧度
    direct: cellOptions.direct + 240,  // 方向
    translate: cellOptions.translate
  });
  */
  
  let squareDirect = cellOptions.direct - 127,
    squareTranslate = cellOptions.translate;
  // 容器大小和方向
  addCSSRule('.hover-square.cell-1', {
    transform: 'rotate(' + squareDirect + 'deg)' + ' ' + squareTranslate + ' !important'
  });
  /*
  // 容器大小和方向
  addCSSRule('.hover-square.cell-2', {
    transform: 'rotate(' + (squareDirect + 120) + 'deg)' + ' ' + squareTranslate + ' !important'
  });
  // 容器大小和方向
  addCSSRule('.hover-square.cell-3', {
    transform: 'rotate(' + (squareDirect + 240) + 'deg)' + ' ' + squareTranslate + ' !important'
  });
  */
}

function blurCell(evt) {
  $('.sector').removeClass('expand');
  document.querySelectorAll('.cell-sector').forEach((item)=>{
    item.classList.remove('expand');
  });

  evt.stopPropagation();
}

function cellClick(dom, evt) {
  let target = evt.target,
    cellIndex = target.getAttribute('cell');
  
  let nodeCtn = $('.node-code').filter(function() {
    return $(this).text() == topovm.selectedCode;
  })[0].parentNode;

  codeCellMap[topovm.selectedCode] = cellIndex;
  // 动态绘制小区信号扇面
  base.index = cellIndex - 0; // 更新信号扇面方向
  
  drawPie({
    selector: base.selector,
    radius: base.radius,     // 外半径
    minRadius: base.minRadius,   // 内半径
    angles: base.angles,  // 弧度
    direct: base.direct*1 + (base.index - 1)*120 // 方向
  });
  // 添加expand类，展示信号区域
  document.querySelectorAll('.cell-sector').forEach((item)=>{
    item.classList.remove('expand');
    item.classList.remove('select');
  });
  let curCell = $('.cell-sector.cell-'+cellIndex, nodeCtn);
  curCell.addClass('expand');
  curCell.addClass('select');
  $('.sector', nodeCtn).addClass('expand');

  evt.stopPropagation();
}
/**
 * @param {Number} radius: 外半径
 * @param {Number} minRadius: 内半径
 * @param {Number} angles: 角度
 * @param {Number} direct: 方向
 **/
  function drawPie({radius, minRadius, angles, direct, selector}) {
  let points = ['50% 50%', '0 0'],
    residue = (angles*1)%45? (angles*1)%45:45,
    percent = 0
    direct = direct*1 || 45;

  angles = angles*1;
  
  angles > 90 && points.push('100% 0');
  angles > 180 && points.push('100% 100%');
  angles > 270 && points.push('0 100%');
  
  //let percent = (100/2) * (Math.tan(2*Math.PI/360 * residue).toFixed(4)); // tan算出来的是相对半径的占比
  
  percent = (100/2) * (Math.tan(2*Math.PI/360 * residue).toFixed(4)); 
  
  if(angles<=45) {
    points.push(percent + '%' + ' 0');
  }else if(angles<=90) {
    points.push(percent + 50 + '%' + ' 0');
  }else if(angles<=135) {
    points.push('100% ' + percent + '%');
  }else if(angles<=180) {
    points.push('100% ' + (percent + 50) + '%');
  }else if(angles<=225) {
    points.push(100 - percent + '%' + ' 100%');
  }else if(angles<=270) {
    points.push(50 - percent + '%' + ' 100%');
  }else if(angles<=315) {
    points.push('0 ' + (100 - percent) + '%');
  }else if(angles<=360) {
    points.push('0 ' + (50 - percent) + '%');
  }
  
  let path = 'polygon(' + points.join(', ') + ')',
    zoom = 5;
  try{
    zoom = globMap.getZoom();
  }catch(e){}

  let rateRels = {
      0: 0.10,
      1: 0.10,
      2: 0.10,
      3: 0.10,
      4: 0.13,
      5: 0.25,
      6: 0.45,
      7: 0.75,
      8: 1.3,
      9: 2,
      10: 2,
      11: 3,
      12: 3,
      13: 5,
      14: 5,
      15: 8,
      16: 8,
      17: 12,
      18: 12
    },
    rate = rateRels[zoom]*zoom;
  console.log('zoom: ', zoom, rate)

  // 外扇面
  addCSSRule(selector + '::before', {
    'clip-path': path
  });
  // 内扇面
  addCSSRule(selector + '::after', {
    'clip-path': path,
    width: minRadius*rate + 'px',
    height: minRadius*rate + 'px',
    top: 'calc(50% - ' + minRadius*rate/2 + 'px)',
    left: 'calc(50% - ' + minRadius*rate/2 + 'px)'
  });
  let rateRadius = radius*rate;
  rateRadius = rateRadius < 30 ? 30:rateRadius;
  // 容器大小和方向
  addCSSRule(selector, {
    top: '-' + rateRadius/2 +'px',
    left: '-' + rateRadius/2 +'px',
    width: rateRadius + 'px',
    height: rateRadius + 'px',
    transform: 'rotate(' + direct + 'deg)'
  });
}
/*
// 绘制扇面
drawPie({
  selector: '.sector',
  radius: 400,     // 外半径
  minRadius: 180,  // 内半径
  angles: 120,     // 弧度
  direct: 45       // 方向
});
*/
function addCSSRule(selector, rules, index) {
    // 创建一个style元素
    let style = document.createElement('style');
    
    // 设置type属性为text/css
    style.type = 'text/css';
    
    // 插入到head中
    document.head.appendChild(style);
    
    // 获取sheet
    let sheet = style.sheet;
    
    // 如果index未提供，则添加到末尾
    index = index || null;
    
    // 如果是CSS规则字符串
    if (typeof rules === 'string') {
        // 直接添加
        sheet.insertRule(selector + ' {' + rules + '}', index);
    } else { // 如果是一个对象
        // 遍历对象中的所有属性
        for (let prop in rules) {
            if (rules.hasOwnProperty(prop)) {
                // 将属性和值转换为字符串
                let rule = prop + ': ' + rules[prop];
                // 添加到样式表中
                sheet.insertRule(selector + ' {' + rule + '}', index);
            }
        }
    }
}

// 扩展字符串的数据映射能力
String.prototype.evaluate = function(map,pKey,ptxt){
  if(pKey) pKey += '.';
  else pKey = '';
  
  let txt = ptxt? ptxt:this.toString();
  for(let key in map){
    if(map.hasOwnProperty(key)){
      if(typeof map[key] == 'object'){
        txt = this.evaluate(map[key],pKey+key,txt);
      }else{
        let reg = new RegExp('{\\s*'+pKey+key+'\\s*}','g');
        txt = txt.replace(reg,map[key]);
      }
    }
  }
  if(!ptxt){
    let rg = new RegExp('{\\s*[a-zA-Z0-9\\.]+\\s*}','g');
    txt = txt.replace(rg,'');
  }
  return txt;
}
// 关闭所有展开的 jQuery EasyUI 下拉框的通用函数
function hideAllComboBoxPanels() {
    // 方法1: 通过 combo-p 面板查找对应的输入框
    $('.combo-p:visible').each(function() {
        let panelId = $(this).attr('id');
        if (panelId) {
            // 查找使用这个面板的 combobox
            $('input.combo-value[combopanelid="' + panelId + '"]').each(function() {
                try {
                    $(this).combo('hidePanel');
                } catch(e) {}
            });
        }
    });
    
    // 方法2: 直接查找所有 combobox/combogrid/combotree 并关闭面板
    $('input.combobox-f,input.combogrid-f,input.combotree-f').each(function() {
        try {
            let panel = $(this).combo('panel');
            if (panel && panel.is(':visible')) {
                $(this).combo('hidePanel');
            }
        } catch(e) {}
    });
    
    // 方法3: 通过类名查找所有可能的 combo 输入框
    $('.combo input.combo-value').each(function() {
        try {
            let panel = $(this).combo('panel');
            if (panel && panel.is(':visible')) {
                $(this).combo('hidePanel');
            }
        } catch(e) {}
    });
}