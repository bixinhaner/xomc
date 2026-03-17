<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ include file="/common/loading.jsp"%>
<style>		
.leftCol{
	margin-left:30px;
	width:230px;
	overflow : inherit;
}
.rightCol{
	margin-left:-30px;
	width:220px;
}
.InfoDivContent{
	height : 90%;
}
.searchTitle{
	color : #b0afba;
	font-size : 10px;
}
#softUpgradeAddOrModify .datagrid-header{
	display:none;
}
.selfConfigSucTip{
	display:inline-block;
	min-width:200px;
	height:38px;
	line-height:38px;
	padding : 0 15px 0 50px;
	margin-left:35px;
	color:#508D9B;
	font-size:16px;
	font-weight:bold;
	background:#E9FBFF url("${ctx}/images/success.png") no-repeat 16px center;
}
.groupInputStyle{
	text-indent:30px;
}
</style>
<div class="slidebarTitleDiv">
	<ul class="slidebarTitleContainer" style="margin-left:0;padding-left:0;">
		<li class="default" id="softUpgradeTitle"></li>
	</ul>
	<div class="tableDiv titleIcon_close" onclick="closeSoftUpdatePlan();" style="position:absolute;right:25px;top:15px;"></div>
</div>
<div class="splitPanel_second" style="width: 100%;height:70%; margin: 15px 0;">
	<div class="leftCol">
		<div class="infoDivTitle"><%=rb.getString("ChanPinLeiXingBiaoZhi")%></div>
		<div class="InfoDivContent" id="productTypeTree">
		</div>
	</div>
	<div class="rightCol" style="width: 60%;">
		<div class="infoDivTitle"><%=rb.getString("ShengJiBanBen")%></div>
		<div class="InfoDivContent" style="padding : 0 10px;">
			<table id="softwareVersion_grid"></table>
		</div>
	</div>
</div>
<div class="linkbuttonGroup" style="margin:30px 70px 20px;">
    <a id="addOrUpdateKPIArithCommit" isBasic="true" isUpdate="true" href="#" class="linkbutton linkbutton_trend" onclick="addOrModifySoftUpgrade(this)" ><span><%=rb.getString("QueDing")%></span></a>
    <a href="#" class="linkbutton linkbutton_nowanna" onclick="closeSoftUpdatePlan()"><span><%=rb.getString("QuXiao")%></span></a>
</div>
<a style="display:none;margin-bottom:20px;margin-left:0px;" class="selfConfigSucTip" id="selfConfigAddSucTip"><%=rb.getString("GuiHuaChuangJianWanCheng")%></a>
<a style="display:none;margin-bottom:20px;margin-left:0px;" class="selfConfigSucTip" id="selfConfigModitySucTip"><%=rb.getString("GuiHuaXiuGaiWanCheng")%></a>
<!-- 软件版本toolbar -->
<div id="toolbar_softwareVersion" class="query_head">
    <div class="queryGroup" style="padding:20px 20px 20px 30px;">
    	<input id="searchSoftVersion" name="search_text" style="" placeholder="<%=rb.getString("ShengJiBanBen")%>">
 		<b class="searchResultImgChangeStyle" onclick="querySoftVersion()"></b>
    </div>
</div>   
<script type="text/javascript">
var selectedVersion = "";
$(function() {
	$("#searchSoftVersion").bind("keyup", function (event) {
        if (event.keyCode == 13) {
            $("#softwareVersion_grid").datagrid("reload");
        }
    });
	closeLoading();
	if(isAdd == "true"){
		$.post("${ctx}/cell/version/getProductType.action", {}, function(data){
			if(data){
				var prodectType = "";
				for(var i=0;i<data.length;i++){
					var nodeTxt = data[i].name;
					var nodeVal = data[i].value;
				    if(nodeTxt.length>11) nodeTxt = nodeTxt.substring(0,10)+'...';
					if(i==0){						
						prodectType += "<a class='groupInputStyle check groupInputStyle_click' oldChecked='false' title='"+nodeTxt+"' onclick='peodectTypeClick(this)' id='" + nodeVal + "'>" + nodeTxt +"</a>";
					}else{
						prodectType += "<a class='groupInputStyle check' oldChecked='false' title='"+nodeTxt+"' onclick='peodectTypeClick(this)' id='" + nodeVal + "'>" + nodeTxt +"</a>";
					}
				}
				$("#productTypeTree").append(prodectType);
				$("#softwareVersion_grid").datagrid({
					url : '${ctx}/cell/halobSelfConfig/getCurrSoftwareVersions.action'
			    })
			}
		},"json");
	}else{
		var selectedRow = $("#softUpgradeDatagrid").datagrid("getSelected");
		var nodeTxt = $("#softUpgradeDatagrid").datagrid("getSelected").product_name;
		var nodeVal = $("#softUpgradeDatagrid").datagrid("getSelected").product_value;
		var prodectType = "<a class='groupInputStyle check groupInputStyle_click' oldChecked='false' title='"+nodeTxt+"' onclick='peodectTypeClick(this)' id='" + nodeVal + "'>" + nodeTxt +"</a>";
		$("#productTypeTree").append(prodectType);	
		$("#softwareVersion_grid").datagrid({
			url : '${ctx}/cell/halobSelfConfig/getCurrSoftwareVersions.action'
	    })
	}
	$("#softwareVersion_grid").datagrid({
		border:false,
   		fit:true,
       	queryParams : {
       		like_fields : 'dest_version'
		},
       	toolbar:'#toolbar_softwareVersion',
       	singleSelect:true,
       	rownumbers:true,
       	fitColumns:true,
       	striped:true,
       	singleSelect : true,
       	idField : 'dest_version',
       	onSelect : onSelect_softwareVersion,
       	onBeforeLoad : onBeforeLoad_softwareVersion,
       	onLoadSuccess: onLoadSuccess_softwareVersion,
       	columns: [[
                   {field: 'operate',width: 50,fixed:true,formatter:function(value, rowData, rowIndex){
	                   	value = '<input type="radio" name="productType"/>';
                	    return value;
                   },align:'center'},
                   {field: 'dest_version',sortable:true, width: 100},
        	]]
	})
})
//新建或修改软件版本升级规划  -- 确定 
function addOrModifySoftUpgrade(){
	var product = $("#productTypeTree .groupInputStyle_click").prop("id");
	var dest_version = $("#softwareVersion_grid").datagrid("getSelected").dest_version;
	var params = {};
	params.product_value = product;
	params.dest_version = dest_version;
	//如果是新建  
	if(isAdd == "true"){
		//新建时需要先判断当前产品类型是否已经存在 
		$.post("${ctx}/cell/halobSelfConfig/isUpgradeRuleExist.action", {product_value : product}, function(data){
			if(data.success){
				$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("DangQianPeiZhiYiCunZaiJiangFuGai")%>", function (r) {
					//已存在，确认覆盖
			        if (r) {
						$.post("${ctx}/cell/halobSelfConfig/saveAutoUpgradeRules.action", params, function(data){
							if(data.success){
								$("#selfConfigAddSucTip").show();
								setTimeout('$("#selfConfigAddSucTip").fadeOut()',1000);
								$("#softUpgradeDatagrid").datagrid("reload");
								setTimeout('closeSoftUpdatePlan()',1000);
							}else{
								$.messager.alert("<%=rb.getString("TiShi")%>", data.message);
					      	    return;
							}
						}, "json");
			        }
			    }).addClass("seriousConfirm");
			}else{
				//不存在，新建
				$.post("${ctx}/cell/halobSelfConfig/saveAutoUpgradeRules.action", params, function(data){
					if(data.success){
						$("#selfConfigAddSucTip").show();
						setTimeout('$("#selfConfigAddSucTip").fadeOut()',1000);
						$("#softUpgradeDatagrid").datagrid("reload");
						setTimeout('closeSoftUpdatePlan()',1000);
					}else{
						$.messager.alert("<%=rb.getString("TiShi")%>", data.message);
			      	    return;
					}
				}, "json");
			}
		}, "json")
	}else{
		//修改
		$.post("${ctx}/cell/halobSelfConfig/saveAutoUpgradeRules.action", params, function(data){
			if(data.success){
				$("#selfConfigModitySucTip").show();
				setTimeout('$("#selfConfigModitySucTip").fadeOut()',1000);
				$("#softUpgradeDatagrid").datagrid("reload");
				setTimeout('closeSoftUpdatePlan()',1000);
			}else{
				$.messager.alert("<%=rb.getString("TiShi")%>", data.message);
	      	    return;
			}
		}, "json");
	}
	
}
function querySoftVersion(){
    $("#softwareVersion_grid").datagrid("reload");
}
//软件版本选中某一行的事件
function onSelect_softwareVersion(param){
	if($("#softwareVersion_grid").datagrid("getSelected")){
		var selectRowVersion = $("#softwareVersion_grid").datagrid("getSelected").dest_version;
		$($("input[name=productType]")[param]).prop("checked",true);
		selectedVersion = selectRowVersion;
	}
	
}
//软件版本加载前事件
function onBeforeLoad_softwareVersion(param){
	var choosedProductTypeId = $("#productTypeTree .groupInputStyle_click").prop("id");
    param["product_value"] = choosedProductTypeId;
	var search_text = $("#searchSoftVersion").val();
    if (search_text != "") {
        param["search_text"] = search_text;
    }
}
//产品类型点击事件
function peodectTypeClick(ele) {
    $("#productTypeTree a").attr("class","groupInputStyle check")
    $(ele).attr('class', 'groupInputStyle check groupInputStyle_click');
    $("#softwareVersion_grid").datagrid("unselectAll");
    $("#softwareVersion_grid").datagrid("reload");
    selectedVersion ="";
}
//版本表格加载成功事件，主要是处理单选按钮的勾选事件
function onLoadSuccess_softwareVersion(value){
    $(this).datagrid("enableContextmenuAutoSize");
    $(this).datagrid("fixRownumber");
    var allData = value.rows;
    if(allData.length>0){
	    if(isAdd == "true"){
		    if(selectedVersion==""){
			    selectedVersion = allData[0].dest_version
		    }
	    }else{
		    for(var i=0;i<allData.length;i++){
			    if(allData[i].checked == "true"){
			    	if(selectedVersion==""){
				    	selectedVersion = allData[i].dest_version;
				    	break;
			    	}
			    }
		    }
	    }
	    for(var i=0;i<allData.length;i++){
		    if(allData[i].dest_version == selectedVersion){
			    $("#softwareVersion_grid").datagrid("selectRow",i);
		    }
	    }
    }
}
</script>