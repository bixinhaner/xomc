<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page import="com.baicells.omc.busi.system.login.entity.UserInfo" %>
<%@page import="com.baicells.omc.busi.utils.ComConstants" %>
<%
	UserInfo ui = (UserInfo) session.getAttribute(ComConstants.SESSION_KEY);
%>

<style>
.logBox{
	position:absolute;
	width:290px;
	height:230px;
	top:34px;
	display:none;
	padding:0 10px 10px 10px;
	-webkit-box-shadow:2px 6px 13px 0 #D8E2F3;
	-moz-box-shadow:2px 6px 13px 0 #D8E2F3;
	box-shadow:2px 6px 13px 0 #D8E2F3;
	z-index:999;
	background:#FFFFFF;
}
.logBox div{
	height:200px;
	overflow-y:auto;
}
.logBox p{
	height:25px;
	padding-top:5px
}
.logBox p span{
	width:20px;
	height:20px;
	float:right;
	cursor:pointer;
	background:url('${ctx}/css/images/sas/sasCloseBox.png') no-repeat right center;
}
.downbg{
	display:inline-block;
	width:20px;
	height:34px;
	cursor:pointer;
	background:url('${ctx}/css/images/sas/sasxiala.png') no-repeat center;
}
</style>
<div>
	<div class="slidebarTitleDiv" style='padding-left:0px;height:50px;'>
		<ul class="slidebarTitleContainer" style='padding-left:0px;'>
			<li class="default"><%=rb.getString("RiZhi") %></li>
		</ul>
		<!-- 导出  -->
		<div class="SASLogExportDiv" style="padding-top:0px;position:absolute;top:28px;right:40px;">
			<span class="SASLogExportImg titleIcon_export iconSize" title='<%=rb.getString("DaoChu")%>' style="padding-left:0px;margin-right:40px" onclick='submitAllSASLog()'></span>
		</div> 
		<div class="tableDiv titleIcon_close" style="position:absolute;right:30px;top:20px;" onclick='closeLogPanel()'></div>
	</div>
	
	<%-- <div class="SASLogExportDiv" style="padding-top:0px;">
		<span class="SASLogExportImg titleIcon_export iconSize" title='<%=rb.getString("DaoChu")%>' style="padding-left:0px;margin-right:40px" onclick='submitAllSASLog()'></span>
	</div>  --%>
	<div id='sasLogDiv' class="singleContentDiv" style="margin-top:65px;flex:1" >		
		<table id="sasLogTable" class="panelTableDiv"></table>
	</div> 
</div>
<%-- 表单-用于导出整体日志 --%>
<form id="formDownloadAllSASLog" style="display:none"></form>
<!-- 工具栏--按CBSD查询  -->
 <div id="toolbar_sasLog" class="toolbarContainer" style="position:relative">
     <div class="queryGroup">
     	<input id='sasLogInput'  placeholder="CBSD">
		<b class="searchResultImgChangeStyle" onclick="$('#sasLogTable').datagrid('reload')"></b>
     </div>
</div> 
<script>
$(function(){
	/* log table 加载 */
	 var SASLogData = {
		 "total":1,
		 "rows":[{
			 direct:'from',
			 object:'SAS',
			 message:'this is a message',
			 cbsds:'1233123123,123123123',
			 time:timeZone
		 }],
		 "footer":[],
		 "properties":null

	 }
	$("#sasLogTable").datagrid({
    	border : false,
        fit : true,
       /*  data:SASLogData, */
        url : '${ctx}/cell/SAS/getMainLog.action',
        rownumbers : true,
        striped : true,
        singleSelect : true,
        fitColumns : true,
        pagination : true,
        queryParams : {},
        pagePosition : 'bottom',
        toolbar:'#toolbar_sasLog',
        idField : 'id',
        columns: [[
				   {field:'id',hidden:true},
                   {field:'direction',width:50,title:'<%=rb.getString("SASFangXiang")%>'},
                   {field:'object',width:50,title:'<%=rb.getString("MuBiao")%>'},
                   {field:'msg',width:100,styler:setPositionStyle,formatter:setcbdsFormat,title:'<%=rb.getString("XiaoXi")%>'},
                   {field:'cbsd',width:100,styler:setPositionStyle,formatter:setcbdsFormat,title:'<%=rb.getString("CBSDSheBeiHao")%>'},
                   {field:'time',sortable:true,width:80,title:'<%=rb.getString("ShiJian")%> (UTC)'},
               ]],
         onBeforeLoad:function(param){
        	 param["cbsd"] = $("#sasLogInput").val();
         },
         onSortColumn:getSASLogSortParams
    });
	 $("#sasLogInput").bind("keyup", function(e){
			if (e.keyCode == 13){
				$('#sasLogTable').datagrid('reload');
			}
		});
})
function setPositionStyle(value,row,index){
	return 'position:relative;';
}
function setcbdsFormat(value,row,index){
	if(value == "" || value == null){
		
	}else{
		var valuestr = value;
		var valueArr = valuestr.split(",");
		value = "<div><div><span style='display:inline-block;width:155px;height:34px;line-height:34px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;'>"+valueArr[0]+"</span><span class='downbg' onclick='slideDownhideBox(this)'></span></div><div class='logBox'><p><span onclick='slideUpShowBox(this)'></span></p><div style='word-break:break-all;white-space:pre-wrap'>"+valuestr+"</div></div></div>"
	}
	return value;
}
/* function setMsgFormatter(value,row,index){
	valueStr = value;
	value = "<div><div><span style='display:inline-block;width:155px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;'>"+valueStr+"</span><span class='downbg' onclick='slideDownhideBox(this)'></span></div><div class='logBox'><p><span onclick='slideUpShowBox(this)'></span></p><div>"+valueStr+"</div></div></div>"
	return value;
} */
var showHideBoxFlag = true;
function slideDownhideBox(e){
	if(!$(e).hasClass("showFlag")){
		$(".logBox").slideUp(500);
		$(e).parent('div').next().slideDown(500);
		var allHeight = $(document).height();
		var tdHeight = $(e).parents("td").offset().top;;
		if((allHeight - tdHeight)<330){
			$(e).parent("div").next().css({"bottom":"34px","top":"unset"});
		}else{
			$(e).parent("div").next().css({"top":"34px","bottom":"unset"});
		};
		$(e).addClass("showFlag")
	}else{
		$(e).parent().next().slideUp(500);
		$(e).removeClass("showFlag");
	}
	
}
function slideUpShowBox(e){
	$(e).parents('.logBox').slideUp(500);
	$(e).parents('.logBox').prev().find(".downbg").removeClass("showFlag");
}
function submitAllSASLog(){
	/* $("#formDownloadAllSASLog").form('submit', {
        url: "${ctx}/cell/SAS/exportMainLog.action",
        onSubmit: function(param){
        	$("#sasLogTable").datagrid("options")
        	param.cbsd = $("#sasLogInput").val();
        	param.sort = sasLogOrderParams.sort?sasLogOrderParams.sort:"";
        	param.order = sasLogOrderParams.order?sasLogOrderParams.order:"";
        }
    }); */
    exportByForm("${ctx}/cell/SAS/exportMainLog.action",{
    	cbsd: $("#sasLogInput").val(),
    	sort: sasLogOrderParams.sort?sasLogOrderParams.sort:"",
    	order: sasLogOrderParams.order?sasLogOrderParams.order:""
    });
}
var sasLogOrderParams = {};
function getSASLogSortParams(sort,order){
	var sortParams = {
			sort:sort || "",
			order:order || ""
	}
	sasLogOrderParams = sortParams
	return sortParams;
}
</script>