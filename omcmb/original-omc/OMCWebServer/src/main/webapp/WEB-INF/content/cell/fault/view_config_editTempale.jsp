<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>
<!-- 告警视图开发页面…… -->
<style>
		/* 新建模板样式 */
	.pciBoxcontainer{
		/* margin:15px 0px 0px 30px; *//*模板中的padding*/
		/* min-height:160px; */
	}
	.baseInfoBox{
		min-height:50px;
		margin-top:20px;
	}
	.baseInfoBox li{
	    width:400px;
	    height:65px;
		float:left;
		margin-left:40px;
	}
	.baseInfoBox li span{
		display:block;
		height:20px;
		line-height:20px;
		color:#85A8BF;
	}
	.baseInfoBox li input{
		padding-left:15px;
		height:26px;
		line-height:26px;
		border:1px solid #85A8BF;
		width:380px;
	}
	.alarmTemSelectDevice{
		
	}
	.alarmTemSelectDevice{
		margin:30px 0 0 40px;
	} 
	.alarmTemSelectDevice span{
		color:#85A8BF;
		margin-right:48px;
	}
	.alarmTemSelectDevice input{
		vertical-align:middle;
		margin-top:-2px;
		margin-bottom:1px;
	}
	.alarmTemSelectDevice input{
		font-size:13px;
	}
	.editSelectedeNBContainer{
		
	}
	#selectDeviceGroup{
		margin:40px 0 0 40px;
	}
	.alarmDeviceGroupTree{
		width:784px;
		height:447px;
		border:1px solid #CCE1EF;
		margin:20px 0 0 40px;
	}
	.alarmTypeTitle{
		min-width:150px;
		height:30px;
		line-height:30px;
		font-size:13px;
		color:red;
		margin-left:55px;
	}
	#selectalarmType{
		margin:20px 0 20px 55px;
	}
	/* 新建模板样式结束 */
</style>

	<!-- 添加模板内容 -->
	<!-- 基本信息 -->
<div id="editTemplateContainer" style="position:absolute;top:15px;overflow:auto;bottom:10px;width:95%;padding-left:60px;">
<div class="alarmViewBaseInfo" style="height:130px">
	<div class="omcPageTitleDiv">
		<ul class="omcPageTitleContainer" style="width:80%">
			<li class="default"><%=rb.getString("JiBenXinXi")%></li>
		</ul>
		<ul class="baseInfoBox">
			<li>
				<span><%=rb.getString("MuBanMingCheng")%></span>
				<input type="text" id="alarmEditTemplateName" onblur="verifyTemName(this)" disabled="disabled"/>
				<span style="color:red"></span>
			</li>
		</ul>
	</div>
</div>
<!-- 响铃选择 -->
<div class="alarmAddBellContainer" style="margin-bottom:25px;">
	<div class="omcPageTitleDiv">
		<ul class="omcPageTitleContainer" style="width:80%">
			<li class="default"><%=rb.getString("XiangLingTiShi")%></li>
		</ul>
		<div class="alarmTemSelectDevice">
			<span><%=rb.getString("XiangLingTiShi")%><%=rb.getString("MaoHao")%></span>
			<input type="radio" name="editringTip" value="openRing" id="editOpenRing">
			<label for="editOpenRing" style="margin-right:80px"><%=rb.getString("KaiQi")%></label>
			<input type="radio" name="editringTip" value="closeRing" id="editCloseRing" checked="checked">
			<label for="editCloseRing"><%=rb.getString("GuanBi")%></label>
		</div>
	</div>
</div>

<!-- 设备选择 -->
<div class="alarmViewSelectDevice">
	<div class="omcPageTitleDiv">
		<ul class="omcPageTitleContainer" style="width:80%">
			<li class="default"><%=rb.getString("SheBeiXuanZe")%></li>
		</ul>
		<div class="alarmTemSelectDevice">
			<span><%=rb.getString("SheBeiLeiXing")%><%=rb.getString("MaoHao")%></span>
			<input type="radio" value="ENB" name="selectDevice" id="editSelecteNB" checked="checked" onclick="editselectedeNB()">
			<label for="editSelecteNB" style="margin-right:80px">eNB</label>
			<input type="radio" value="EPC" name="selectDevice" id="editSelectEPC" onclick="editselectedEPC()" class="CODE_EPC hidden">
			<label id="editEpclabel" for="editSelectEPC" class="CODE_EPC hidden" style="margin-right:80px">EPC</label>
			<input type="radio" value="EGW" name="selectDevice" id="editSelectEGW" onclick="editselectedEGW()" class="CODE_EGW hidden">
			<label id="editEgwlabel" for="editSelectEGW" class="CODE_EGW hidden">EGW</label>
		</div>
		<!-- enb设备组 -->
		<div id="editSelectedeNBContainer" class="editSelectedeNBContainer">
		<div id="editSelectDefaultGroup" class="alarmTypeTitle"></div>
				<!-- 搜索框 -->
			<div id="selectDeviceGroup" class="queryGroup">
				<input id="editAlarmTreesetSearchText" placeholder='<%=rb.getString("SheBeiZuMingCheng")%>'/>
				<b onclick="filteralarmEditDeviceGroup()"></b>
			</div>
			<!-- 设备组树 -->
		<!-- 	<div class="alarmDeviceGroupTree"></div> -->

			<div id='alarmEditDeviceGroupDiv' class="tree-lines" style='padding-left:10px;padding-top:10px;user-select:none;height:440px;width:777px;border:1px solid #CCE1EF;margin-top:10px;margin-left:40px;overflow:auto'>
				<ul id=alarmEditDeviceGroup></ul>
			</div>

		</div>
	</div>
</div>

	<!-- 告警标识 -->
<div class="alarmViewType" style="margin-top:40px;">
	<div class="omcPageTitleDiv">
		<ul class="omcPageTitleContainer" style="width:80%">
			<li class="default"><%=rb.getString("XuanZeGaoJing")%></li>
		</ul>
	</div>
	<!-- 未选择告警标识的时候 提示文字 -->
	<div id="editSelectAlarmType" class="alarmTypeTitle"></div>
		<!-- 搜索框 -->
	<div id="toolbar_editSelectalarmType" class="queryGroup" style="margin:0 0 20px 55px;">
		<input id="editAlarmTypesetSearchText" placeholder="<%=rb.getString("GaoJingWeiYiBiaoZhi")%>/<%=rb.getString("YanZhongChengDu")%>"/>
		<b onclick="javascript: $('#editAlarmTypeTable').datagrid('reload');"></b>
	</div>
	<div class="alarmTypeTableContainer" style="width:788px;height:390px;border:1px solid #CCE1EF;margin-left:55px;">
		<table id="editAlarmTypeTable"></table>
	</div>
</div>

<div class="linkbuttonGroup" style="margin:40px 0 20px 55px;">
       	<a href="#" class="linkbutton linkbutton_trend" onclick="editAlarmTemplate()"><span><%=rb.getString("QueDing")%></span></a>
       	<a href="#" class="linkbutton linkbutton_nowanna" onclick="cancleAlarmTemplate()"><span><%=rb.getString("QuXiao")%></span></a>
</div>

</div>
<!-- 添加模板结束 -->
<script type="text/javascript">
var templateId = "${templateId}";
var templateName = "${templateName}";
var tempDeviceType = "${deviceType}"
var tempAlarmPE = "${alarmPromptEnable}"

$(function(){
	closeLoading();
	
	//判断为QB版本时，显示是否启用告警响铃的功能 
	if (isQb == 'true'){
		$(".alarmAddBellContainer").show();
	}else{
		$(".alarmAddBellContainer").hide();
	}
	
	//判断是否支持EPC，若支持则显示EPC设备类型的单选按钮
	if(isSupportEPC == 'true'){
		$("#editSelectEPC").show();
		$("#editEpclabel").show();
	}else{
		$("#editSelectEPC").hide();
		$("#editEpclabel").hide();
	}
	
	if(isSupportEGW == 'true'){
		$("#editSelectEGW").show();
		$("#editEgwlabel").show();
	}else{
		$("#editSelectEGW").hide();
		$("#editEgwlabel").hide();
	}
	/* 编辑告警搜索框回车事件 */
	$("#editAlarmTypesetSearchText").bind("keyup", function(e){
		if (e.keyCode == 13){
			$('#editAlarmTypeTable').datagrid('reload');
		}
	});
	 /* 查看设备组树搜索框回车事件 */
	$("#editAlarmTreesetSearchText").bind("keyup", function(e){
		if (e.keyCode == 13){
			filteralarmEditDeviceGroup()
		}
	});
	$("#alarmEditTemplateName").val(templateName);
	$('#alarmEditDeviceGroup').tree({
		url: '${ctx}/fault/viewConfig/getDeviceGroupTree.action', 
		/* data: dataSource,  */
		checkbox:true,
		queryParams:{templateId:templateId},
		idField: 'id',            //定义关键字段来标识树节点。也就是数据的id
		treeField: 'text',        //定义树形显示字段
		fitColumns:true,
		animate:true,
		});
	//根据选择模板判断响铃告警开启还是关闭
	/* if(tempAlarmPE == "Y"){
		$("#editOpenRing").attr("checked","checked");
	}else{
		$("#editCloseRing").attr("checked","checked");
	} */
	//根据选择模板判断cpe 还是enb
	if(tempDeviceType == "ENB"){
		$("#editSelecteNB").attr("checked","checked");
		$("#editSelectEGW").removeAttr("checked");
		$("#editSelectEPC").removeAttr("checked");
		$("#editSelectedeNBContainer").css("display","block");
	}else if(tempDeviceType == "EGW"){
		$("#editSelectEGW").attr("checked","checked");
		$("#editSelecteNB").removeAttr("checked");
		$("#editSelectEPC").removeAttr("checked");
		$("#editSelectedeNBContainer").css("display","none");
	}else{
		$("#editSelectEPC").attr("checked","checked");
		$("#editSelectEGW").removeAttr("checked");
		$("#editSelecteNB").removeAttr("checked");
		$("#editSelectedeNBContainer").css("display","none");
	}
	/* 告警标识初始化 */
	$("#editAlarmTypeTable").datagrid({
		url: '${ctx}/fault/viewConfig/queryAlarmIdentifierPageList.action',
		queryParams:{templateId:templateId,like_fields:"alarm_identifier,alarm_serverity",deviceType:tempDeviceType},
		fit:true,
		fitColumns:true,
		border:false,
		rownumbers:true,
		idField:'ALARM_IDENTIFIER',
		/* toolbar:'#toolbar_editSelectalarmType', */
		striped: true,
	    pagination : true,
	    pagePosition : 'bottom',
	    onBeforeLoad : editAlarmTypeTableBeforeLoad,
	    onLoadSuccess :getcheckedRow,
	    onCheck:addCheckeAlarmIcentifier,
	    onUncheck:delCheckeAlarmIcentifier,
	    onUncheckAll:cleanHasChoseAlarm,
	    onCheckAll:choseAllEditAlarm,
		columns: [[
			/* {field: 'template_id', hidden: true}, */
			{field: 'ck',checkbox:true},
			{field: 'DEVICE_TYPE_NAME', width:50,sortable:true,formatter:formatDiviceType, title:'<%=rb.getString("SheBeiLeiXing")%>'},
			{field: 'ALARM_IDENTIFIER',width:100,sortable:true, title: '<%=rb.getString("GaoJingWeiYiBiaoZhi")%>'},
			{field: 'propable_cause',width:100,sortable:false,title: '<%=rb.getString("KeNengYuanYin")%>'},
			{field: 'ALARM_SERVERITY',formatter:formatOper,width:100,sortable:true,title: '<%=rb.getString("YanZhongChengDu")%>' },
			<%-- {field: 'alarmSetOperation',width:100,fixed:true,formatter:alarmSetFormatter,title:'<%=rb.getString("CaoZuo")%>'},    --%>
		]],
		
	});

	
})
function verifyTemName(e){
	var templateName = $(e).val();
	if(templateName == ""){
		$(e).next().html('<%=rb.getString("MuBanMingChengWeiKong")%>');
	}else{
		$(e).next().html('');
	}
}

function editselectedeNB(){
	var choseenb = $("#editSelecteNB").val();
	params = {like_fields:"alarm_identifier,alarm_serverity",deviceType:choseenb,templateId:templateId};
	$("#editAlarmTypeTable").datagrid('clearChecked');
	$("#editAlarmTypeTable").datagrid("reload",params)
	$("#editSelectedeNBContainer").slideDown(500);
}
function editselectedEPC(){
	var chosecpe = $("#editSelectEPC").val();
	params = {like_fields:"alarm_identifier,alarm_serverity",deviceType:chosecpe,templateId:templateId};
	$("#editAlarmTypeTable").datagrid('clearChecked');
	$("#editAlarmTypeTable").datagrid("reload",params)
	$("#editSelectedeNBContainer").slideUp(500);
}
function editselectedEGW(){
	var choseegw = $("#editSelectEGW").val();
	params = {like_fields:"alarm_identifier,alarm_serverity",deviceType:choseegw,templateId:templateId};
	$("#editAlarmTypeTable").datagrid('clearChecked');
	$("#editAlarmTypeTable").datagrid("reload",params)
	$("#editSelectedeNBContainer").slideUp(500);
}
//确定修改模板
function editAlarmTemplate(){
	var params = {};
	/* 模板名称不可修改 */
	params.templateId = templateId;
	params.templateName = templateName;
	/* var ringTip = $("input:radio[name='editringTip']:checked").val();
	if(ringTip == "openRing"){
		params.alarmPromptEnable = "Y";
	}else{
		params.alarmPromptEnable = "N";
	} */
	params.alarmPromptEnable = "N";
	var deviceType = $("input:radio[name='selectDevice']:checked").val();
	params.deviceType = deviceType
	if(deviceType == "ENB"){
		var deviceGroup = [];
		var addTreenodes = $("#alarmEditDeviceGroup").tree('getChecked');
		
	 	if(addTreenodes.length == 0){
	 		$("#editSelectDefaultGroup").html('<%=rb.getString("QingXuanZeSheBeiZu")%>');
	 		$("#editTemplateContainer").animate({
				scrollTop:220
			}); 
			return ;
		}else{
			$("#editSelectDefaultGroup").html('');
			deviceGroup = addTreenodes.map(function(x){
				return x.id;
			});
			for(var i = 0;i<deviceGroup.length;i++){
				if(deviceGroup[i] == 0){
					deviceGroup.splice(i,1);
				}
			}
			params.deviceGroup = deviceGroup;
		}
	}else{
		var deviceGroup = [];
		params.deviceGroup = deviceGroup;
	}
	/* var alarmIdentifier = [];
	var addAlarmTable = $("#editAlarmTypeTable").datagrid("getChecked");
	alarmIdentifier = addAlarmTable.map(function(x){
		return x.ALARM_IDENTIFIER;
	}) */
	
	if(editTemplateCheckArr.length == 0){
		$("#editSelectAlarmType").html('<%=rb.getString("GaoJingWeiYiBiaoZhi")%>');
		return ;
	}else{
		$("#editSelectAlarmType").html('');
		params.alarmIdentifier = editTemplateCheckArr;
	}
	$.ajax({
		type:'post',
		url:'${ctx}/fault/viewConfig/updateViewConfig.action',
		data:params,
		dataType:"json",
		traditional:true,
		success:function(data){
			if(data["success"] == true){
				$("#alarmSetTable").datagrid("reload");
				  cancleAlarmTemplate()
			}else{
				 showMsg('error_msg',data["message"]);
			}
		}
	})	
}
//编辑模板 告警标识 加载前函数
function editAlarmTypeTableBeforeLoad(param) {
	var searchCellCode = $("#editAlarmTypesetSearchText").val();
	if (searchCellCode) {
		param["search_text"] = searchCellCode;
	}
}
function filteralarmEditDeviceGroup(){
	var searchgroup = $("#editAlarmTreesetSearchText").val();
	 $('#alarmEditDeviceGroup').tree('doFilter',searchgroup);
}
function getcheckedRow(data){
	var checkedrow = data.rows;
	for(var i = 0;i<checkedrow.length;i++){
		if(checkedrow[i].checked == true){
			$("#editAlarmTypeTable").datagrid("checkRow",i)
		}
	}
}
//勾选将新勾选的告警标识的放入editTemplateCheckArr数组
function addCheckeAlarmIcentifier(index,rowData){
	var hasCheckedAlarmId = rowData.ALARM_IDENTIFIER;
	//判断这个是否在数组中存在 -1不存在  不存在 加入
	var flagNum = $.inArray(hasCheckedAlarmId,editTemplateCheckArr);
	if(flagNum == -1){
		editTemplateCheckArr.push(hasCheckedAlarmId);
	}
}
//取消勾选
function delCheckeAlarmIcentifier(index,rowData){
	var hasCheckedAlarmId = rowData.ALARM_IDENTIFIER;
	//判断这个是否在数组中存在  存在删除
	var flagNum = $.inArray(hasCheckedAlarmId,editTemplateCheckArr);
	if(flagNum != -1){
		editTemplateCheckArr.removeArrElement(hasCheckedAlarmId);
	}
}
//取消全选  将editTemplateCheckArr清空
function cleanHasChoseAlarm(rows){
	editTemplateCheckArr = [];
}
function choseAllEditAlarm(){
	var alarmIdentifier = [];
	var addAlarmTable = $("#editAlarmTypeTable").datagrid("getChecked");
	alarmIdentifier = addAlarmTable.map(function(x){
		return x.ALARM_IDENTIFIER;
	});
	editTemplateCheckArr = alarmIdentifier;
}
</script>