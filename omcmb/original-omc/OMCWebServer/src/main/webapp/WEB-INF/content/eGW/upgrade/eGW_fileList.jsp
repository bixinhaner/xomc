
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
	.editegwUpgradeFileDiv,.viewegwUpgradeFileDiv{
	position:absolute;
	width:900px;
	right:-1000px;
	background:#fff;
	top:0px;
	bottom:0px;
	z-index:100;
	box-shadow:2px 3px 16px rgba(158,200,222,0.5);
	overflow:auto;
	border:1px solid #E4E7EC;
}
	.egwUpgradeImportDiv{
		width:453px;
		height:623px;
		position:absolute;
		background:#fff;
		border:1px solid #E4E7EC;
		right:-500px;
		box-shadow:2px 3px 16px rgba(158,200,222,0.5);
		z-index:100;
		top:73px;
	}
	.showegwFileOp{
		position:absolute;
		z-index:950;
		left:10px;
		width:160px;
		background:#fff;
		border:1px solid rgba(188,188,188,0.1);
		display:none;
		box-shadow:0 5px 15px #d8d8d8;
	}
	.showegwFileOp div{
		width:110px;
		height:40px;
		line-height:40px;
		padding-left:14px;
		padding-right:36px;
		display:block;
		margin:0 !important;
		border-bottom:1px solid #e5f0f6;
		color:#000;
		background-position-x:15px !important;
	}
	.showegwFileOp div:hover{
		background:#EDF6FF;
	}
	.showegwFileOp span{
		font-size:12px;
		margin-left:14px;
	}
	.showegwFileOp div:last-of-type {
		border:none;
	}
	.slideDownAllDiv{
		position:absolute;
		top: 0;
		left: 0;
		bottom: 0;
		width:100%;
		height:100%;
		background:#fff;
		z-index:100;
		border:none;
		display:none;
	}
</style>
<%-- 版本文件管理页面 --%>
<div class="panelDefault" style='overflow:hidden'>
	<!-- 右上角导入按钮 -->
	<div style='right: 115px;' class="circleIcon placeholder-bt CODE_EGW hidden" placeholder="<%=rb.getString("DaoRuWenJian")%>">
		<span class="el-icon el-icon-circle-import" onclick="importegwUpgradeFile()"></span>
	</div>
	
	<div class="circleIcon placeholder-bt CODE_EGW hidden" style="right: 65px;" placeholder="<%=rb.getString("XinJianRenWu")%>">		
		<span class="el-icon el-icon-circle-addTask" onclick="toAddEGW()"></span>
	</div>
	<div class="circleIcon placeholder-bt" style='right:15px;' placeholder="<%=rb.getString("RenWuLieBiao")%>">		
		<span class="el-icon el-icon-circle-taskList" onclick="toViewEGW()"></span>
	</div>
	
	<div id="eGWSoftOperateDiv" class="slideDownAllDiv">
		<div id="egw_slide_content"></div>
	</div>
	<div class='tabsTitle'>
		<span class='active'><%=rb.getString("ShengJi")%></span>
	</div>
	<div class="contentDiv enbUpgradeDiv" style="top:30px;">	
		<table id="egwFileUpgradeTable"></table>
	</div>	
	<div class='viewegwUpgradeFileDiv'></div>
	<div class='editegwUpgradeFileDiv'></div>
	<div class='egwUpgradeImportDiv'></div>
</div>
<%-- 下载文件的用的表单 --%>
<form method="post" style="display: none"  id="egwdownloadFileForm"></form>

<script type="text/javascript">
// 引入 eGW_taskList.jsp 任务列表页面  根据bool visible判断是新建还是查看
function slideEGWDiv(bool,visible){
	if(bool == true) {
		var url = '${ctx}/egw/softwareUpgrade/toSoftwareUpgrade.action',
			$conDiv = $('#egw_slide_content');
		$conDiv.html('');
		$conDiv.load(url,function(data){
			$.parser.parse(this);
			if(visible == true) hiddenEgwTable();
		});
		$("#eGWSoftOperateDiv").slideDown(500,function(){
			$conDiv.resize();
		});
	}else {
		$("#eGWSoftOperateDiv").slideUp(500);
	}
}
// 新建任务按钮
function toAddEGW(){
	slideEGWDiv(true,true);
}
// 查看任务列表
function toViewEGW(){
	slideEGWDiv(true);
}
$(function() {
    $("#egwFileUpgradeTable").datagrid({
    	url:'${ctx}/egw/softwareFile/querySoftwareFilePageList.action',
    	queryParams:{timeZone:timeZone},
    	fit:true,
    	fitColumns:true,
    	border:false,
    	singleSelect:true,
    	rownumbers:true,
    	striped:true,
    	pagination:true,
    	pagePosition:'bottom',
    	idField:'id',
    	columns: [[
				{field: 'id',hidden:true},
				{field:'file_name',hidden:true},
 				{field: 'operation',width:30,fixed:true,styler:setStyle,formatter:moreFileFormatter,title:''},
 				{field: 'version',width:200,title:'<%=rb.getString("BanBen") %>'},
 				{field: 'product_type',width:150,title:'<%=rb.getString("ChanPinLeiXingBiaoZhi") %>'},
 				{field: 'file_size',width:100,title:'<%=rb.getString("WenJianDaXiao") %>'},
 				{field: 'upload_time',width:150,title:'<%=rb.getString("ShangChuanShiJian") %>'},
    			]]
    })
	$(document).click(function(e){
		 var e = e || window.event;
	        var elem = e.target || e.srcElement;
	        while(elem){
	            if($(elem).hasClass('el-icon-operation-more') || elem.className == 'showegwFileOp'){
	                return
	            }
	            elem = elem.parentNode;
	        }
	        $(".showegwFileOp").css('display','none');
	});
})

/**
* 表格 更多 格式化
* @param value{string} 绑定值
* @param rowData{object}  行数据
* @param rowIndex{number}  下标
*/
function moreFileFormatter(value,rowData,rowIndex){
	 var XiaZai = "<%=rb.getString("XiaZai")%>";
	var XinXi = "<%=rb.getString("XinXi")%>";
	var XiuGai = "<%=rb.getString("XiuGai")%>";
	var ShanChu = "<%=rb.getString("ShanChu")%>"
	var row_id = rowData.id;
	var fileName = rowData.file_name;
	if(value == null){
		value = "<div class='el-icon el-icon-operation-more' title='"+ CaoZuo+"' onclick='openMoreOperation("+ row_id +",this)' ></div>";
		var opt="";
		opt = opt + "<div class='el-icon el-icon-operation-info' onclick='viewegwFile(\"" + row_id + "\")'><span>"+XinXi+"</span></div>";
		opt = opt + "<div class='el-icon el-icon-operation-edit CODE_EGW hidden' style='margin-right:0px' onclick='editegwFile(\"" + row_id + "\")'><span>"+XiuGai+"</span></div>";
		opt = opt + "<div class='el-icon el-icon-operation-download' onclick='egwfileLibraryDownload(\"" + row_id + "\",\"" + fileName + "\")'><span>"+XiaZai+"</span></div>";
		opt = opt + "<div class='el-icon el-icon-operation-delete CODE_EGW hidden' onclick='delegwFile(\"" + row_id + "\")'><span>"+ShanChu+"</span></div>";
		value = value + "<div class='showegwFileOp'>"+ opt +"</div>";
	}
	return value; 
}
// 点击打开更多
function openMoreOperation(idVal,e){
	var thisTop = $(e).offset().top;
	var allHeight = $(document).height();
	var indexRow = $("#egwFileUpgradeTable").datagrid("getRowIndex",idVal);
	var rowHeight = $(".enbUpgradeDiv .datagrid-view2").find("tr[datagrid-row-index="+indexRow+"]").height();
	if((allHeight - thisTop) < 240){
		$(e).next(".showegwFileOp").css("bottom",rowHeight+"px");                                                                                                                                                                                                                                                             
	}else{
		$(e).next().css("top",(rowHeight)+"px");
	}
	var current = $('.showegwFileOp',$(e).parent());
	$('.showegwFileOp').not(current).hide();
	current.fadeToggle(300);
}
//设置操作列单元格样式 
function setStyle(){
	return 'position:relative';
}
// 查看
function viewegwFile(idVal){
	cancelImportegwFile();
	closeEditegwFileWindow();
	$(".showegwFileOp").hide();
	$(".viewegwUpgradeFileDiv").animate({right:'0px'},400,function(){
		$(".viewegwUpgradeFileDiv").panel({
			width:900,
			queryParams:{
				id:idVal
			},
			href:'${ctx}/egw/softwareFile/toSoftwareFileView.action'
		})
	});
}
//关闭查看弹窗
function closeViewegwFileWindow(){
	$(".viewegwUpgradeFileDiv").animate({right:'-1000px'},400);
}
// 导入
function importegwUpgradeFile(){
	closeViewegwFileWindow();
	closeEditegwFileWindow();
	$(".egwUpgradeImportDiv").animate({right:'0px'},400);
	$(".egwUpgradeImportDiv").panel({
		width:453,
		href:'${ctx}/egw/softwareFile/toSoftwareFileImport.action'
	})
}
// 关闭导入
function cancelImportegwFile(){
	$(".egwUpgradeImportDiv").animate({right:'-500px'},400);
}
// 修改
function editegwFile(idVal){
	cancelImportegwFile();
	closeViewegwFileWindow();
	$(".showegwFileOp").hide();
	$(".editegwUpgradeFileDiv").animate({right:'0px'},400,function(){
		$(".editegwUpgradeFileDiv").panel({
			width:900,
			queryParams:{
				id:idVal
			},
			href:'${ctx}/egw/softwareFile/toSoftwareFileEdit.action'
		})
	});
}
// 关闭修改
function closeEditegwFileWindow(){
	$(".editegwUpgradeFileDiv").animate({right:'-1000px'},400,function(){
		$(".editegwUpgradeFileDiv").html("");
	});
}
// 下载
function egwfileLibraryDownload(idVal,fileName){
	/* $("#egwdownloadFileForm").form('submit', {
        url: "${ctx}/egw/softwareFile/downloadSoftwareFile.action",
        onSubmit: function(param){
            param.id = idVal;
            param.fileName = fileName;
			var bool = checkParams(param)
			if(!bool) return false;
        }
    }); */
    exportByForm("${ctx}/egw/softwareFile/downloadSoftwareFile.action",{
    	id: idVal,
    	fileName: fileName
    })
}
// 删除
function delegwFile(idVal){
    $.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingShanChuWenJian")%>", function(r) {
        if (r) {
            $.post("${ctx}/egw/softwareFile/delSoftwareFileInfos.action", {id: idVal}, function(data) {
                if (data["success"]) {
                     $("#egwFileUpgradeTable").datagrid("reload");
                } else {
                    showMsg('error_msg',data["message"]);
                }
            }, "json");
        }
    }).addClass("seriousConfirm");
}
</script>