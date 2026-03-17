<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>

<style type="text/css">
.inputslist li{
	margin:0;
}
.textbox.combo{
	vertical-align:top;
}
.titleButtonText {
	transition: opacity 0.5s ease-in;
}
#operCpeProTypeDiv{
	width:878px;
	/* height:94%; */
	position:absolute;
	top:0px;
	bottom:0px;
	background:#fff;
	right:-900px;
	overflow:hidden;
	z-index:100;
}
#toolbar_cpeProType .textbox.combo{
	vertical-align:top;
}
.inputslist label {
    margin: 0px 8px 0px 0px;
}
#cpeProTypeAdvDiv ul li{
	float: left;
    height: 35px;
    margin-right:80px;
    margin-bottom:30px;
    margin-top:10px;
}
.defaultQuery{
	padding:10px 0px;
}
.highQueryArrow span{
	vertical-align:super;
}
.tabsTitle{
	border:none;
}
</style>

<%--cpe产品类型页面   --%>
<div class="panelDefault" style="overflow:hidden">
	 <!-- 添加产品类型按钮 -->
    <div class="addConfig circleIcon" id="addCpeProType"> 
    	<span id='cpeProCircleButton' class="el-icon el-icon-circle-add" onclick="addCpeProType()"></span>
    	<div class="titleButtonText"><%=rb.getString("TianJia")%></div>
    </div>
    <div class="singleTitle">
		<span class="active"><%=rb.getString("ProductType")%></span>	
	</div>
	<div class="singleContentDiv">
		<table id="cpeProductTypeTable"></table>
	</div>
	
	<!-- 操作产品类型 -->
    <div id="operCpeProTypeDiv" class='slidebarPanel'></div>
</div>
    
<%-- 工具栏 --%>
<div id="toolbar_cpeProType" class="admin_query_head">
    <form id="cpeProTypeQueryForm">
    	<div class="easyui-query" tips="<%=rb.getString("GaoJiChaXun") %>" 
    		name="searchText" 
    		inputId="cpeProTypeSearInput" targetId="cpeProTypeAdvDiv" 
    		placeholder="<%=rb.getString("ChanPinLeiXingBiaoZhi") %>" 
    		data-options="query: vagueCpeProTypeQuery">
		</div>
        <div id="cpeProTypeAdvDiv" class="advanceQuery_content">
	        <ul class="inputslist">
	            <li>
	            	<label><%=rb.getString("ChanPinLeiXingBiaoZhi") %><%=rb.getString("MaoHao")%></label><br>
	        		<input type="text" id="proTypeInput" name="productName" class="border-box border" style="width:200px;"/>
	            </li>
	            <li>
	                <label><%=rb.getString("CanShuMoXing") %><%=rb.getString("MaoHao")%></label><br>
	                <select id="cpeProTypeParaModel" class="easyui-combobox border border-box" data-options="editable:false" name="modelId" style="height:26px;width:200px;"></select>
	            </li>
	            <li>
	            	<label><%=rb.getString("ChuangJianZhe")%><%=rb.getString("MaoHao")%></label><br>
	            	<input type="text" id="cpeProTypeUserCode" class="border-box border" name="create_user" style="height:26px;width:200px;"/>
	            </li>
	            <li id='cpeProTypeStartTimeDiv'>
	                <label><%=rb.getString("KaiShiShiJian")%><%=rb.getString("MaoHao")%></label><br>
	                <div>	                
		                <input id="cpeProTypeStartTime" class="easyui-datetimebox border-box border" data-options="editable:false" name="start_time" style="height:26px;width:200px;">
	                </div>
	            </li>
	            <li id='cpeProTypeEndTimeDiv'>
	                <label><%=rb.getString("JieShuShiJian")%><%=rb.getString("MaoHao")%></label><br>
	                <div>	                
		                <input id="cpeProTypeEndTime" class="easyui-datetimebox border-box border" data-options="editable:false" name="end_time" style="height:26px;width:200px;">
	                </div>
	            </li>
	        </ul>
	        <div class="linkbuttonGroup">
	        	<a href="#" class="linkbutton linkbutton_trend" onclick="accurateCpeProTypeQuery()"><span><%=rb.getString("ChaXun")%></span></a>
	        	<a href="#" class="linkbutton linkbutton_nowanna" onclick="resetQueryInput()"><span><%=rb.getString("ChaXunChongZhi")%></span></a>
        	</div>
        </div>
    </form>
</div>

<script type="text/javascript">
var globalQueryParams = {
		TimeZone : timeZone,
        // 拼装查询框模糊匹配的字段（注：要和数据库表字段一致）
        like_fields : 'PRODUCT_NAME' 
    };
// 产品类型列表 参数模型图标筛选
var rules = [
			{
			title: '<%=rb.getString("CanShuMoXing")%>',
			match: function(code){
				return code == 'MODEL_NAME';
			},
			action: function(code,e){
				var key = 'modelId';
					/* 配置筛选菜单可选项，可以通过接口获取数据 */
					var data = [];
					var params = $('#cpeProductTypeTable').datagrid('options').queryParams;
					$.post("${ctx}/cell/CPE/getModelNameForDatagridFilterCommbox.action",globalQueryParams,function(json){
						json.map(function(item,index){
							var row = {name:'MODEL_NAME',label:item.model_name,value:item.id};
							data.push(row);
						})
						data.map(function(item){
							if(params[key]){
								var vals = params[key].split(',');
								if(vals.includes(item.value+'')) item.checked = true;
							}else{
								item.checked = true;
							}
						});
						var sidValue = $('#cpeProTypeParaModel').combobox('getValue');
						if(sidValue){/* 原该查询参数有值，筛选项唯一且不可操 */
							data = data.filter(function(item){
								var bool = (item.value == sidValue);
								if(bool) item.disabled = true;
								return bool;
							});
						}
						/* 生成筛选菜单 */
						filterMenu({
						data: data,
						fn: function(tips){
								tips.css({left:e.x-$('#menuAnimate').width(),top:110});
								$('#mainpage').append(tips);
						},
						click: function(values){
							params[key] = values;
							$('#cpeProductTypeTable').datagrid('reload');
						}
						});
					},"json")
			}
			}
];
$(function() {
    closeLoading();
    $.post("${ctx}/cell/CPE/getModelNameForCommbox.action","",function(data){
    	var arrData = [ {
    		id:"",
    		model_name:'<%=rb.getString("QuanBu")%>'
    	}]
    	data.map(function(item,index){
    		arrData.push(item);
    	})
    	$("#cpeProTypeParaModel").combobox({
    		valueField:'id',
    		textField:'model_name'
    	}) 
    	$("#cpeProTypeParaModel").combobox("loadData",arrData)
    },"json")
    var paramData = {
    	timeZone:0,
    	like_fields:"PRODUCT_NAME",
    	page:1,
    	rows:20
    }
    $.post("${ctx}/cell/CPE/getCpeProductModelList.action",paramData,function(data){
    	if(data.total == 0){
			showMsg('prompt_msg',"<%=rb.getString("ChanPinLeiXingWeiKongTiShi")%>");
		}
    },"json")
    $("#cpeProductTypeTable").datagrid({
		url:'${ctx}/cell/CPE/getCpeProductModelList.action',
		queryParams : {
            timeZone : timeZone,
            like_fields : 'PRODUCT_NAME'
        },
		singleSelect:true,
		fit:true,
		fitColumns:true,
		border:false,
		rownumbers:true,
		pagePosition:'bottom',
		pagination: true,
		striped: true,
		toolbar:"#toolbar_cpeProType",
		onLoadSuccess:function(data){
			$(this).datagrid("fixRownumber");
			$(this).datagrid("enableContextmenuAutoSize");
		},
		onBeforeLoad:beforeload_cpeProType,
		columns: [[
			{field: 'PRODUCT_ID',hidden:true},
			{field: 'MODEL_ID',hidden:true,},
			{field: 'PRODUCT_NAME',sortable:true,width:100,title:'<%=rb.getString("ChanPinLeiXingBiaoZhi") %>'},
			{field: 'MODEL_NAME',width:100,title:titleFilter},
			{field: 'CREATE_USER',width:100,title:'<%=rb.getString("ChuangJianZhe")%>',formatter:cpeProTypeUserCodeFmt},
			{field: 'CREATE_TIME',sortable:true,width:100,title:'<%=rb.getString("ChuangJianShiJian")%>'},
			{field: 'operation',width:100,title:'<%=rb.getString("CaoZuo") %>',formatter:cpeProTypeOperFmt}
		]]
	});
    <%-- 搜索回车 --%>
	$("#cpeProTypeSearInput").bind("keyup", function(e){
		if (e.keyCode == 13){
			vagueCpeProTypeQuery();
		}
	});
	$('#mainpage').mousedown(function(){/* 菜单隐藏处理 */
		try{
		    var target = event.target, list = Array.from(target.classList),
		        plist = Array.from(target.parentNode.classList);
		    if(!(list.includes('filter-menu') || list.includes('filter-item') || plist.includes('filter-item'))){
		      $('.filter-menu').hide();
		    }
		}catch(e){}
	});
});
/**
*  表格数据 操作 格式化数据
* @param value{string}   绑定值
* @param rowData{object}   行数据
* @param rowIndex{number}   下标
*/ 
function cpeProTypeOperFmt(value, rowData, rowIndex){
	var modelId = rowData.MODEL_ID;
	var productId = rowData.PRODUCT_ID;
	var productName = rowData.PRODUCT_NAME;
	var value = "";
	value = "<div class='el-icon el-icon-operation-view' title='<%=rb.getString("ChaKan") %>' style='display:inline-block;cursor:pointer;' onclick='viewCpeProType()'></div>";
	value += "<div class='el-icon el-icon-operation-delete' title='<%=rb.getString("ShanChu") %>' style='margin-left:15px;display:inline-block;cursor:pointer;'onclick='deleteCpeProType(\""+modelId+"\",\""+productId+"\",\""+productName+"\")'></div>";
	return value;
}
/**
*  表格数据 创建者 格式化数据
* @param value{string}   绑定值
* @param rowData{object}   行数据
* @param rowIndex{number}   下标
*/ 
function cpeProTypeUserCodeFmt(value,rowData,rowIndex){
	if(value == "null"){
		return "";	
	}else{
		return value;
	}
}

//新建产品类型
function addCpeProType(){
	$(".titleButtonText").html("<%=rb.getString("GuanBi")%>");
	$(".circleBg").removeClass("add_circle").addClass("close_circle addCpeProTypeFlag");
	$('#operCpeProTypeDiv').animate({right:"0px"},450);
	$('#operCpeProTypeDiv').panel({
		href:"${ctx}/cell/CPE/toCpeModel.action",
		queryParams:{
			type:'add'
		},
		width:878
	})
}
// 查看产品类型
function viewCpeProType(){
	$('#operCpeProTypeDiv').animate({right:"0px"},450);
	$('#operCpeProTypeDiv').panel({
		href:"${ctx}/cell/CPE/toCpeModel.action",
		queryParams:{
			type:'view'
		},
		width:878
	})
}
function cancelOperCpeProType(){
	$(".titleButtonText").html("<%=rb.getString("TianJia")%>");
	$(".circleBg").addClass("add_circle").removeClass("close_circle addCpeProTypeFlag viewCpeProTypeFlag");
	$('#operCpeProTypeDiv').animate({right:"-900px"},300);
}
function showTipCircle(ele){
		$(ele).next().css('opacity','1');
}

function hideTipCircle(ele){
		$(ele).next().css('opacity','0');
}
// 请求之前 赋值操作
function beforeload_cpeProType(param){
	param.timeZone = timeZone;
}
/**
*  删除产品类型
* @param modelId{number}    参数类型
* @param productId{object}   产品类型id
* @param productName{string}   产品类型名称
*/ 
function deleteCpeProType(modelId,productId,productName){
	$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueRenShanChuChanPinLeiXing")%>", function (r) {
        if (r) {
        	var params = {
        			"modelId":modelId,
        			"productId":productId,
        			"productName":productName,
        			"type":"delete"
        	}
        	$.post("${ctx}/cell/CPE/saveOrUpdateProductAndModelInfo.action",params,function(data){
                 if (data["success"]) {
                	$("#cpeProductTypeTable").datagrid("reload");
                } else {
                    showMsg('error_msg',data.message);
                    return;
                } 
            }, "json");
        }
    }).addClass("seriousConfirm");
}
//下拉列表展开
function moreQuerySlideFun(){
	if($("#cpeProTypeMoreQueryImg").attr("flag")=="1"){
		$("#cpeProTypeAdvDiv").slideDown(500);
		$("#cpeProTypeMoreQueryImg").attr("flag","0");
		$("#cpeProTypeMoreQueryImg").addClass('expanded');
	}else{
		$("#cpeProTypeAdvDiv").slideUp(400);
		$("#cpeProTypeMoreQueryImg").attr("flag","1");
		$("#cpeProTypeMoreQueryImg").removeClass('expanded');
	}	
}
//高级查询结果
function vagueCpeProTypeQuery(){
	$("#cpeProTypeUserCode").val("");
	$("#proTypeInput").val("");
	$("#cpeProTypeParaModel").combobox('setValue', '');
	$("#cpeProTypeStartTime").datetimebox('setValue', null);
	$("#cpeProTypeEndTime").datetimebox('setValue', null);
    var queryParamObj = $('#cpeProTypeQueryForm').serializeJson();
    queryParamObj.timeZone = timeZone;
    queryParamObj['like_fields'] = 'PRODUCT_NAME';
    globalQueryParams = $.extend({},queryParamObj);
    doSearchUrl('cpeProductTypeTable', queryParamObj, '${ctx}/cell/CPE/getCpeProductModelList.action');
    
	$("#cpeProTypeAdvDiv").slideUp(100);	
	$("#cpeProTypeMoreQueryImg").removeClass('expanded');
	$("#cpeProTypeMoreQueryImg").attr("flag","1"); 
	$("#cpeProType_hiddenSpan").hide();
}
// 高级查询确定
function accurateCpeProTypeQuery(){
	$("#cpeProTypeSearInput").val("");
	var dataStartTime = $("#cpeProTypeStartTime").datetimebox("getValue");
	var dataEndTime = $("#cpeProTypeEndTime").datetimebox("getValue");
    //开始时间不能晚于结束时间
    var validTimeResult = validateStartAndStopTime(dataStartTime, dataEndTime);
    if ("false" == validTimeResult) {
        showMsg('prompt_msg',"<%=rb.getString("JieShuShiJianBuNengXiaoYuKaiShiShiJian")%>");
        return;
    }
    var queryParamObj = $('#cpeProTypeQueryForm').serializeJson();
    queryParamObj.timeZone = timeZone;
    queryParamObj['like_fields'] = 'PRODUCT_NAME';
    globalQueryParams = $.extend({},queryParamObj);
    doSearchUrl('cpeProductTypeTable', queryParamObj, '${ctx}/cell/CPE/getCpeProductModelList.action');
    
	$("#cpeProTypeAdvDiv").slideUp(100);	
	$("#cpeProTypeMoreQueryImg").removeClass('expanded');
	$("#cpeProTypeMoreQueryImg").attr("flag","1"); 
	$("#cpeProType_hiddenSpan").hide();
}
//高级查询重置
function resetQueryInput(){
	$("#cpeProTypeUserCode").val(null);
	$("#proTypeInput").val(null);
	$("#cpeProTypeParaModel").combobox('setValue', '');
	$("#cpeProTypeStartTime").datetimebox('setValue', null);
	$("#cpeProTypeEndTime").datetimebox('setValue', null);
}
// 验证开始时间和结束时间是否合法
function validateStartAndStopTime(startTimeStr, endTimeStr){
    if (isNotNull(startTimeStr) && isNotNull(endTimeStr)) {
        var startDate = dateParser(startTimeStr);
        var endDate = dateParser(endTimeStr);
        if (startDate.getTime() < endDate.getTime()) {
            return "true";
        }
    }
    if (!isNotNull(startTimeStr) && isNotNull(endTimeStr)) {
        return "true";
    }
    if (isNotNull(startTimeStr) && !isNotNull(endTimeStr)) {
        return "true";
    }
    if(""== startTimeStr&& "" == endTimeStr){
    	return "true";
    }
    return "false";
}

function isNotNull(arg){
	if(arg == null){
		return false;
	}
	if(arg == ""){
		return false;
	}
	return true;
}
</script>