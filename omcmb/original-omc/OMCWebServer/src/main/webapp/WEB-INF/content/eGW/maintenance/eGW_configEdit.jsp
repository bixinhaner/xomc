<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page import="com.baicells.omc.busi.system.login.entity.UserInfo" %>
<%@page import="com.baicells.omc.busi.utils.ComConstants" %>
<%
	UserInfo ui = (UserInfo) session.getAttribute(ComConstants.SESSION_KEY);
%>

<style>
	.configConatiner{
	
		position:absolute;
		background:#FFFFFF; 
		left:15px;
		right:15px;
		bottom:15px;
		top:10px;
		padding:20px 0 0 50px;
		z-index:90;
		overflow-y:auto;
		overflow-x:hidden;
	}
	.eGWConfigInfoItemDiv{
		/* display:inline-block; */
		width:441px;
		margin:5px 0px 0 30px;
		vertical-align:top;
	}
	
	.eGWConfigInfoItemDiv label{
		display:block;
		line-height:25px;
		color:#797979;
	}
	.eGWConfigInfoItemDiv input{
		width:396px !important;
	}
	.eGWConfigInfoItemDiv img{
		vertical-align:middle;
	}
	.eGWConfigInfoItemDiv .prompt{
		display:block;
		line-height:25px;
		height:25px;
		color: red;
	}
	.eGWConfigInfoItemDiv select{
		height:27px;
		width:396px;
	}
	.bottomLine{
		height:2px;
		width:880px;
		background:#DCECF7;
		margin-left:30px;
	}
	.addIp div{
		display:inline-block;
		height:19px;
		line-height:19px;
		background:#E6F5FA;
		border:1px solid #CAE4F7;
		width:140px;
		margin-top:4px;
		margin-right:10px;
		padding-left:10px;
	}
	.addIp div img{
		vertical-align:middle;
		float:right;
		margin-right:4px;
		cursor:pointer;
	}
	.addIp div b{
		display:none;	
	}
	.neweNBContainer{
		width:920px;
		position:fixed;
		top:50px;
		right:-1500px;
		background:#FFFFFF;
		z-index:100;
		border:1px solid #E4E7EC;
		box-shadow:-10px 0px 10px rgba(0,0,0,0.15);
		bottom:15px;
		z-index:100;
	}
	.neweNBConfigDiv{
	}
	.neweNBConfigDiv .eGWConfigInfoItemDiv{
		/* display:inline-block; */
		width:400px;
		margin:5px 0px 0 20px;
		vertical-align:top;
	}
	
	.neweNBConfigDiv .eGWConfigInfoItemDiv label{
		display:block;
		line-height:25px;
		color:#797979;
	}
	.neweNBConfigDiv .eGWConfigInfoItemDiv input{
		width:350px !important;
	}
	.neweNBConfigDiv .eGWConfigInfoItemDiv img{
		vertical-align:middle;
	}
	.neweNBConfigDiv.eGWConfigInfoItemDiv .prompt{
		display:block;
		line-height:25px;
		height:25px;
		color: red;
	}
	.neweNBConfigDiv .eGWConfigInfoItemDiv select{
		/* height:27px;
		width:396px; */
	}
	.addLinkDiv{
		position:absolute;
		/* height:240px; */
		background:#FFFFFF; 
		top:55px;
		right:0px;
		left:0px;
		bottom:0px;
		z-index:999;
		padding-top:33px;
		display:none;
		padding-left:20px;
		box-shadow:2px 6px 13px 0 rgba(207,215,232,0.5);
	}
	.addLinkDiv .addLinkConfigInfoItemDiv{
		display:inline-block; 
		width:330px;
		margin:5px 0px 0 30px;
		vertical-align:top;
	}
	.addLinkDiv .addLinkConfigInfoItemDiv label{
		display:block;
		line-height:25px;
		color:#797979;
	}
	.addLinkDiv .addLinkConfigInfoItemDiv input{
		width:300px !important;
	}
	.addLinkDiv .addLinkConfigInfoItemDiv img{
		vertical-align:middle;
	}
	.addLinkDiv .addLinkConfigInfoItemDiv .prompt{
		display:block;
		line-height:25px;
		height:25px;
		color: red;
	}
	.collectingLogs{
		float:left;
		min-width:200px;
		height:38px;
		line-height:38px;
		/* text-align:center; */
		text-indent:50px;
		margin:30px 0 0 35px;
		color:#508D9B;
		font-size:16px;
		font-weight:bold;
		background:#E9FBFF url("${ctx}/images/success.png") no-repeat 16px center;
	}
	.el-card__body{
		display:flex;
		flex-direction:column;
		flex:1 1 auto;
		height:100%;
		overflow:auto;
	}
</style>

<div class="configConatiner">
<!-- 右上角关闭按钮 -->
<div class="omcTitleButton" style="top:90px;position:fixed;right:50px;">
	<span class="titleButtonText"><%=rb.getString("GuanBi")%></span>
	<span class="el-icon el-icon-circle-close" onmouseenter="showTipText(this)" onmouseleave="showTipText(this)" onclick="closeeGWConfigDiv()"></span>
</div>  
	<div class="basicConfig">
		<div class="group-title not-extend"><span class="title-icon"></span><span class="title-text"><%=rb.getString("JiBenPeiZhi")%></span></div>
		<div class="eGWConfigInfoItemDiv">
			<label for="eGWplmn">PLMN</label>
		    <input id="eGWplmn" type="text" value="" maxlength="40" class="easyui-validatebox border border-box" style="width:300px;" onblur="testPlmn(this)"/>
		    <span id="" style="color:#9AADB9;" class="prompt" ><%=rb.getString("QingTianRuWuDaoLiuWeiShuZi")%></span>
		</div>
		<div class="eGWConfigInfoItemDiv">
			<label for="trafficName">eGW Mode</label>
		    <select id="eGWMode" class="easyui-combobox border border-box" style="" data-options="editable:false,onSelect:chooseeGWMode">
		    	<option value="0">level1</option>
		    	<option value="1">level2</option>
		    </select>
			<span id="" class="prompt" ></span>
		</div>
		<div class="egwModelevel2" style="display:none;">
			<div class="eGWConfigInfoItemDiv" style="display:inline-block;">
				<label for="eGWUplinkeGWIp">UpLink SIAP to eGW IP</label>
			    <input id="eGWUplinkeGWIp" type="text" value="" maxlength="40" class="easyui-validatebox border border-box" style="width:300px;" onblur="testTpAddr(this)"/>
			    <span id="Traffic_PCC_NAMECheckSpan" class="prompt" ></span>
			</div>
			<div class="eGWConfigInfoItemDiv" style="display:inline-block;">
				<label for="eGWUplinkMMEIp">UpLink SIAP to MME IP</label>
			    <input id="eGWUplinkMMEIp" type="text" value="" maxlength="40" class="easyui-validatebox border border-box" style="width:300px;" onblur="testTpAddr(this)"/>
			    <span id="Traffic_PCC_NAMECheckSpan" class="prompt" ></span>
			</div>
		</div>
		<div class="eGWConfigInfoItemDiv" style="display:inline-block;">
			<label for="eGWMMEIPeNB">S1-MME IP to eNB</label>
		    <input id="eGWMMEIPeNB" type="text" value="" maxlength="40" class="easyui-validatebox border border-box" style="width:300px;" onblur=""/>
		    <span id='addeGWMMEIPeNB' class='el-icon el-icon-plus' onclick="addeGWIP(this)"></span>
		    <div class="addIp">
		    	<%-- <div class="addIpDiv">
		    		<span>192.168.102.11</span><img src="${ctx}/skin/${manufacturer}/images/newIcon/operatorIcon/operationIcon_delete.png">
		    	</div> --%>
		    </div>
		    <span id="" class="prompt" ></span>
		</div>
		<div class="eGWConfigInfoItemDiv" style="display:inline-block;">
			<label for="eGWMMEPorteNB">S1-MME port to eNB</label>
		    <input id="eGWMMEPorteNB" type="text" value="" maxlength="40" class="easyui-validatebox border border-box" style="width:300px;" disabled="disabled" onblur="testPort(this)"/>
		    <span id="Traffic_PCC_NAMECheckSpan" class="prompt" ></span>
		</div>
		<div class="eGWFTUPDiv" style='min-height:130px'>
			<div class="eGWConfigInfoItemDiv" style="display:inline-block;">
				<label for="GWUplinkGTPU">eGW uplink GTPU IP</label>
			    <input id="GWUplinkGTPU" type="text" value="" maxlength="40" class="easyui-validatebox border border-box" style="width:300px;" onblur=""/>
			    <span id='addeGWUplinkGTPU' class='el-icon el-icon-plus' onclick="addeGWIP(this)"></span>
			    <div class="addIp"></div>
			    <span id="" class="prompt" ></span>
			</div>
			<div class="eGWConfigInfoItemDiv" style="display:inline-block;">
				<label for="GWUpDownGTPU">eGW downlink GTPU IP</label>
			    <input id="GWUpDownGTPU" type="text" value="" maxlength="40" class="easyui-validatebox border border-box" style="width:300px;" onblur=""/>
			    <span id='addeGWUpDownGTPU' class='el-icon el-icon-plus' onclick="addeGWIP(this)"></span>
			    <div class="addIp"></div>
			    <span id="" class="prompt" ></span>
			</div>
		</div>
		<!-- 底边线 -->
		<div class="bottomLine"></div>
	</div>
	<!-- eNB Config -->
	<div class="eNBConfig" style="margin-top:30px;min-width:800px;max-width:880px;">
		<div class="group-title not-extend"><span class="title-icon"></span><span class="title-text"><%=rb.getString("JiZhanPeiZhi")%></span></div>
		<span style="color:#cc0000;margin-left:20px"><%=rb.getString("ZhiShaoXuYaoTianJiaYiTiaoeNBConfig")%></span>
		<span style="float:right;" class='el-icon el-icon-plus' onclick="addeNBConfig()"></span>
		<!-- enb 表格 -->
		<div style="margin-left:30px;height:300px;">
			<table class="easyui-datagrid" id="addeNBConfigTable" fit="true" data-options="fitColumns:true,singleSelect:false,border:true,
                    rownumbers:true,striped:true,singleSelect:true">
				<thead>
				<tr>
				    <th data-options="field:'index',hidden:true" width="80">index</th>
					<th data-options="field:'enb_id'" width="150">eNodeB ID</th>
					<th data-options="field:'tac'" width="150">TAC</th>
					<th data-options="field:'enbConfigOperator',formatter:eNBConfigFormatter" width="80">Operation</th>
				</tr>
				</thead>
		    </table>
		</div>
		
		<!-- 底边线 -->
		<div class="bottomLine" style="margin-top:30px;margin-bottom:30px;"></div>
	</div>

	<!-- advanced config -->
	<div class="advancedConfig">
		<div class="group-title not-extend"><span class="title-icon"></span><span class="title-text"><%=rb.getString("GaoJiPeiZhi")%></span></div>
		<div class="">
			<div class="eGWConfigInfoItemDiv" style="display:inline-block;">
				<label for="eGWAdvancecFlex">S1-Flex Enable</label>
			    <select id="eGWAdvancecFlex" class="easyui-combobox border border-box" style="" data-options="editable:false">
			    	<option value="0">&nbsp;</option>
			    	<option value="1">ENABLE</option>
			    	<option value="2">DISABLE</option>
			    </select>
			    <span id="Traffic_PCC_NAMECheckSpan" class="prompt" ></span>
			</div>
			<div class="eGWConfigInfoItemDiv" style="display:inline-block;">
				<label for="eGWAdvanceLog">Log Level</label>
				<select id="eGWAdvanceLog" class="easyui-combobox border border-box combobox-f combo-f textbox-f" style="" data-options="editable:false">
			    	<option value="0">DISABLE</option>
			    	<option value="1">DEBUG</option>
			    	<option value="2">INFO</option>
			    	<option value="3">ALL</option> 
			    	<option value="4">WARNING</option>
			    	<option value="5">ERROR</option>
			    </select>
			    <span id="Traffic_PCC_NAMECheckSpan" class="prompt" ></span>
			</div>
		</div>
	</div>
	
	<!-- advanced config 结束 -->
	
	<a id="addNewEnbSaveBtn" class="linkbutton" style="margin-top:30px;margin-bottom:30px;margin-left:30px;float:left;" onclick="addNewEnbSave()"><span><%=rb.getString("BaoCun")%></span></a>
	<div id="saveenbConfigBtn" style="display:none" class="collectingLogs">修改enb成功</div>
	<%-- <a id="editNewEnbSaveBtn" class="linkbutton" style="margin-top:30px;margin-bottom:30px;" onclick="editNewEnbSave()"><span><%=rb.getString("BaoCun")%></span></a> --%>
	<!-- new eNB-->
	<div class="neweNBContainer" style="overflow-y:auto;display:flex;flex-direction:column">
		<div class='el-card__header'>
			<span id='neweNBTitle'></span>
			<span class="el-icon el-icon-close" style="position:absolute;right:30px;top:0px;" onclick='eNBConfigClose()'></span>
			<span id="linkGoback" class="tableDiv el-icon el-icon-goback form_bt_rebaack show" style="position:absolute;right:50px;top:0px;display:none" onclick='addLinkClose()'></span>
		</div>
		<div class='el-card__body'>
					<!-- 编辑    添加 link结束 -->
			<div class="neweNBConfigDiv" style="background:#fff;border:1px solid #EEE">
				<div class="basicConfig" style='margin-left:20px;margin-top:20px;'>
					<h5><%=rb.getString("JiZhanPeiZhi")%></h5>
					<div class="level2">
						<div class="eGWConfigInfoItemDiv" style="display:inline-block;">
							<label for="basicEnbId">eNodeB ID</label>
						    <input id="basicEnbId" type="text" value="" maxlength="40" class="easyui-validatebox border border-box" disabled="disabled" style="width:300px;" onblur="testeNBId(this)"/>
						    <span id="Traffic_PCC_NAMECheckSpan" class="prompt" ></span>
						</div>
						<div class="eGWConfigInfoItemDiv" style="display:inline-block;">
							<label for="basicTac">TAC</label>
						    <input id="basicTac" type="text" value="" maxlength="40" class="easyui-validatebox border border-box" style="width:300px;" onblur="testBasicTac(this)"/>
						    <span id="Traffic_PCC_NAMECheckSpan" class="prompt" ></span>
						</div>
					</div>
				</div>
					<!-- 底边线 -->
				<div class="bottomLine" style="width:795px;margin:30px 0px 30px 20px;"></div>
				<!-- 表格 -->
				<div class="basicConfig" style="width:812px;height:450px;position:relative;margin-left:20px;">
					<h5 style="display:inline-block"><%=rb.getString("LianLu")%></h5>
					<span style="color:#cc0000;margin-left:20px"><%=rb.getString("LinkMessageTit")%></span>
					<span style="float:right;" class='el-icon el-icon-plus' onclick="addLink('add')"></span>
					<!-- 编辑    添加 link -->
					<%-- <div class="addLinkDiv">
						<input id="addLinkEnbId" value="" style="display:none"/>
						<div class="addLinkConfigInfoItemDiv">
							<label for=addEdIteNBIp>eNB_IP</label>
						    <input id="addEdIteNBIp" type="text" value=""  class="easyui-validatebox border border-box" style="width:300px;" onblur="testTpAddr(this)"/>
						    <span id="" class="prompt" ></span>
						</div>
						<div class="addLinkConfigInfoItemDiv">
							<label for=addEdIteNBPort>eNB_Port</label>
						    <input id="addEdIteNBPort" type="text" value=""  class="easyui-validatebox border border-box" style="width:300px;" onblur="testPort(this)"/>
						    <span id="" class="prompt" ></span>
						</div>
						<div class="addLinkConfigInfoItemDiv">
							<label for=addEdItMMEIp>MME_IP</label>
						    <input id="addEdItMMEIp" type="text" value=""  class="easyui-validatebox border border-box" style="width:300px;" onblur="testTpAddr(this)"/>
						    <span id="" class="prompt" ></span>
						</div>
						<div class="addLinkConfigInfoItemDiv">
							<label for=addEdItMMEPort>MME_PORT</label>
						    <input id="addEdItMMEPort" type="text" value=""  class="easyui-validatebox border border-box" style="width:300px;" onblur="testPort(this)"/>
						    <span id="" class="prompt" ></span>
						</div>
						<div class="linkbuttonGroup" style="margin:15px 0px 0px 30px;">
							<a id="saveAddLink" class="linkbutton" onclick="saveAddLink()"><span><%=rb.getString("BaoCun")%></span></a>
							<a id="saveEditLink" class="linkbutton" onclick="saveEditLink()"><span><%=rb.getString("BaoCun")%></span></a>
							<a class="linkbutton linkbutton_nowanna" onclick="addLinkClose()"><span><%=rb.getString("QuXiao")%></span></a> 
						</div>
					</div> --%>
					<!-- 编辑    添加 link结束 -->
					<div style="margin-left:20px;height:300px;">
						<table id="addLinkConfigTable" class="panelTableDiv"></table>
					</div> 
					
				</div>
			</div>
		</div>
		<div class='el-card__footer'>
			<div class="windowButtonGroup" style="float:left !important;margin-top:10px;">	
				<a id="saveNewEnbBtn" class="linkbutton" style="margin-bottom:0px;margin-right:0px;" onclick="saveNewEnb()"><span><%=rb.getString("BaoCun")%></span></a>
				<a id="saveEditEnbBtn" class="linkbutton" style="margin-bottom:0px;" onclick="saveeditNewEnb()"><span><%=rb.getString("BaoCun")%></span></a>
			</div>	
			<div id="saveEnbsuccessTit" style="display:none" class="collectingLogs">修改enb成功</div>
		</div>
		<!-- 编辑    添加 link -->
				<div class="addLinkDiv">
					<input id="addLinkEnbId" value="" style="display:none"/>
					<div class="addLinkConfigInfoItemDiv">
						<label for=addEdIteNBIp>eNB_IP</label>
					    <input id="addEdIteNBIp" type="text" value=""  class="easyui-validatebox border border-box" style="width:300px;" onblur="testTpAddr(this)"/>
					    <span id="" class="prompt" ></span>
					</div>
					<div class="addLinkConfigInfoItemDiv">
						<label for=addEdIteNBPort>eNB_Port</label>
					    <input id="addEdIteNBPort" type="text" value=""  class="easyui-validatebox border border-box" style="width:300px;" onblur="testPort(this)"/>
					    <span id="" class="prompt" ></span>
					</div>
					<div class="addLinkConfigInfoItemDiv">
						<label for=addEdItMMEIp>MME_IP</label>
					    <input id="addEdItMMEIp" type="text" value=""  class="easyui-validatebox border border-box" style="width:300px;" onblur="testTpAddr(this)"/>
					    <span id="" class="prompt" ></span>
					</div>
					<div class="addLinkConfigInfoItemDiv">
						<label for=addEdItMMEPort>MME_PORT</label>
					    <input id="addEdItMMEPort" type="text" value=""  class="easyui-validatebox border border-box" style="width:300px;" onblur="testPort(this)"/>
					    <span id="" class="prompt" ></span>
					</div>
					<div class="linkbuttonGroup" style="margin:15px 0px 0px 30px;">
						<a id="saveAddLink" class="linkbutton" onclick="saveAddLink()"><span><%=rb.getString("BaoCun")%></span></a>
						<%-- <a id="saveEditLink" class="linkbutton" onclick="saveEditLink()"><span><%=rb.getString("BaoCun")%></span></a> --%>
						<a class="linkbutton linkbutton_nowanna" onclick="addLinkClose()"><span><%=rb.getString("QuXiao")%></span></a> 
					</div>
				</div>
		
	</div>
	<!-- enb config结束 -->
</div>
<script>
var addLinkParams = [];
var editLinkParams = [];
var delLinkParams = [];

var s1MmeIpToEnbDel = [];
var egwUplinkGtpuIpDel = [];
var egwDownlinkGtpuIpDel = [];

var s1MmeIpToEnbADD = [];
var egwUplinkGtpuIpADD = [];
var egwDownlinkGtpuIpADD = [];
$(function(){

/* 	var tabobj = '{"total":2,"rows":[{"eNB_ID":"111","TAC":"a1,b2,c3"},{"eNB_ID":"112","TAC":"a2"}]}';
	var datas = $.parseJSON(tabobj);  */ 
	//渲染enbconfig表格
	$("#addeNBConfigTable").datagrid({data:editConfigData.eNB_Config});
	//渲染数据
	$("#eGWplmn").val(editConfigData.plmn);
	$("#eGWUplinkeGWIp").val(editConfigData.upLinkS1APToEgwIp);
	$("#eGWUplinkMMEIp").val(editConfigData.upLinkS1APToMMEIp);
	setTimeout(function(){
		if(editConfigData.upLinkS1APToEgwIp != "" || editConfigData.upLinkS1APToMMEIp != ""){
			$("#eGWMode").combobox('setValue','1');
			$(".egwModelevel2").show();
			
		}else{
			$("#eGWMode").combobox('setValue','0');
			$(".egwModelevel2").hide();
		}
	},0)
	
	if(editConfigData.s1MmePortToEnb == ""){
		$("#eGWMMEPorteNB").val("36412")
	}else{
		$("#eGWMMEPorteNB").val(editConfigData.s1MmePortToEnb);
	}
	
	//渲染s1-mmeIp to enb
	var slMMEIpArr = editConfigData.s1MmeIpToEnb;
	for(var i=0;i<slMMEIpArr.length;i++){
		var addIpBox="<div class='addIpDiv'><span>"+slMMEIpArr[i]+"</span><img src='${ctx}/css/images/newIcon/operatorIcon/operationIcon_delete.png' onclick='deleteThisIP(this)'/><b>"+slMMEIpArr[i]+",</b></div>"
		$("#addeGWMMEIPeNB").next(".addIp").append(addIpBox);
	}
	//渲染eGW uplink GTUP ip
	var eGWUplinkGTUPIpArr = editConfigData.egwUplinkGtpuIp;
	for(var i=0;i<eGWUplinkGTUPIpArr.length;i++){
		var addIpBox="<div class='addIpDiv'><span>"+eGWUplinkGTUPIpArr[i]+"</span><img src='${ctx}/css/images/newIcon/operatorIcon/operationIcon_delete.png' onclick='deleteThisIP(this)'/><b>"+eGWUplinkGTUPIpArr[i]+",</b></div>"
		$("#addeGWUplinkGTPU").next(".addIp").append(addIpBox);
	}
	//渲染eGW downling GTUP ip
	var eGWDownGTUPIpArr = editConfigData.egwDownlinkGtpuIp;
	for(var i=0;i<eGWDownGTUPIpArr.length;i++){
		var addIpBox="<div class='addIpDiv'><span>"+eGWDownGTUPIpArr[i]+"</span><img src='${ctx}/css/images/newIcon/operatorIcon/operationIcon_delete.png' onclick='deleteThisIP(this)'/><b>"+eGWDownGTUPIpArr[i]+",</b></div>"
		$("#addeGWUpDownGTPU").next(".addIp").append(addIpBox);
	}
	
	
	
	setTimeout(function(){
		//渲染s1-flex enable
		$("#eGWAdvancecFlex").combobox('setValue',editConfigData.s1FlexEnable);
		//渲染log level
		$("#eGWAdvanceLog").combobox('setValue',editConfigData.logLevel);
	},0);
    
})

//添加ip地址并校验	
var editconfigTabNum = -1;

function addeGWIP(e){
	var ipValue = $(e).prev('input').val();
	//判断 inarray 是否在数组中存在
	
	if(ipValue == ""){
		$(e).siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDeIpDiZhi")%>');
		return;
	}
	var hasIp = $(e).next('.addIp').children('.addIpDiv').find('b').text();
	hasIp = hasIp.substring(0,hasIp.length-1)
	var hasIpArr = hasIp.split(',');
	var isExist = $.inArray(ipValue,hasIpArr);
	if(isExist == -1){//meiyou
		
	}else{
		$(e).siblings('.prompt').text('<%=rb.getString("IPDiZhiYiCunZai")%>');
		return;
	}
	
	var ipValNum = $(e).next('.addIp').find('.addIpDiv').length;
	if($(e).attr("id") == "addeGWMMEIPeNB"){
		if(ipValNum >= 4){
			$(e).siblings('.prompt').text('<%=rb.getString("BuNengChaoGuoSiTiaoIP")%>');
			return;
		}
	}else{
		if(ipValNum >= 8){
			$(e).siblings('.prompt').text('<%=rb.getString("BuNengChaoGuoBaTiaoIP")%>');
			return;
		}
	}
	
	//判断输入ip格式
	var ipReg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
	if(ipReg.test(ipValue)){
		if(removeIp(ipReg,ipValue)){
			$(e).siblings('.prompt').text('');
			var addIpDiv="<div class='addIpDiv'><span>"+ipValue+"</span><img src='${ctx}/css/images/newIcon/operatorIcon/operationIcon_delete.png' onclick='deleteThisIP(this)'/><b>"+ipValue+",</b></div>"
			$(e).next('.addIp').append(addIpDiv);
			$(e).prev('input').val("");
		}else{
			$(e).siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDeIpDiZhi")%>');
			return;
		}	
	} else{
		$(e).siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDeIpDiZhi")%>');
		return;
	}
	//判断分别加入各个添加的ip数组
	var typeOfIp = $(e).siblings("input").attr("id");
	if(typeOfIp == "eGWMMEIPeNB"){
		s1MmeIpToEnbADD.push(ipValue);
	}else if(typeOfIp == "GWUplinkGTPU"){
		egwUplinkGtpuIpADD.push(ipValue);
	}else if(typeOfIp == "GWUpDownGTPU"){
		egwDownlinkGtpuIpADD.push(ipValue);
	}
}

//删除一个IP地址
function deleteThisIP(e){
	var typeIpStr = $(e).parents('.addIpDiv').parents('.addIp').siblings("input").attr("id");
	var typeIpStrVal = $(e).siblings("span").text();
	if(typeIpStr == "eGWMMEIPeNB"){
		var mmeipArr = editConfigData.s1MmeIpToEnb;
		var mmeIpisExist = $.inArray(typeIpStrVal,mmeipArr);
		if(mmeIpisExist == -1){//meiyou
			//不在后台传过来的数据中就是在新添加的数组中  在新添加 数组中删除的时候 从添加的数组中剔除
			s1MmeIpToEnbADD.removeArrElement(typeIpStrVal);
		}else{
			s1MmeIpToEnbDel.push(typeIpStrVal);
		}
	}else if(typeIpStr == "GWUplinkGTPU"){
		var egwUpLinkipArr = editConfigData.egwUplinkGtpuIp;
		var uplinkIpisExist = $.inArray(typeIpStrVal,egwUpLinkipArr);
		if(uplinkIpisExist == -1){//meiyou
			egwUplinkGtpuIpADD.removeArrElement(typeIpStrVal);
		}else{
			egwUplinkGtpuIpDel.push(typeIpStrVal);
		}
		
	}else if(typeIpStr == "GWUpDownGTPU"){
		var egwDownLinkipArr = editConfigData.egwDownlinkGtpuIp;
		var downlinkIpisExist = $.inArray(typeIpStrVal,egwDownLinkipArr);
		if(uplinkIpisExist == -1){//meiyou
			egwDownlinkGtpuIpADD.removeArrElement(typeIpStrVal);
		}else{
			egwDownlinkGtpuIpDel.push(typeIpStrVal);
		}
		
	}
	var addipDivLen = $(e).parents('.addIpDiv').parents('.addIp').children().length;
	if(addipDivLen == 1){
		$(e).parents('.addIpDiv').parents('.addIp').siblings('.prompt').text('<%=rb.getString("BuNengShaoYuYiTiaoIP")%>');
	}
	$(e).parents('.addIpDiv').remove();
}
function successTable(){
	
}
//添加一条eNB Config
function addeNBConfig(){
	$("#basicEnbId").siblings('.prompt').text('');
	$("#basicTac").siblings('.prompt').text('');
	var enbConfigNum = $("#addeNBConfigTable").datagrid('getRows').length;
	if(enbConfigNum >=32){
		showMsg('prompt_msg','<%=rb.getString("HongZhanSheZhiZuiDaXianDu")%>');
		return;
	}
	editconfigTabNum = -1;
	$("#saveNewEnbBtn").css('display','inline-block');
	$("#saveEditEnbBtn").css('display','none'); 
	$("#basicEnbId").val("");
	$("#basicTac").val("");
	$("#neweNBTitle").text('<%=rb.getString("XinJianJiZhan")%>');
	//enbid  可用
	$("#basicEnbId").removeAttr("disabled");
	$(".neweNBContainer").animate({right:'20px'});
	 /* var linktabobj = {"total":2,"row":[{enbIp:"111",enbPort:"27",mmeIp:"192.167.1.1",mmePort:"89"},{enbIp:"222",enbPort:"89",mmeIp:"192.11.1.1",mmePort:"89"}]};  */
	 /*var datas = $.parseJSON(linktabobj);     	
	$("#addLinkConfigTable").datagrid({data:datas}); */
	 $("#addLinkConfigTable").datagrid({
	    	/* url:'${ctx}/egw/config/queryEgwPageList.action', */  
	    	data:[], 
	    	border: true,
	    	fit: true,
	    	queryParams : {
	            timeZone : timeZone
	        },
	    	rownumbers:true,
	    	fitColumns: true,
	        striped: true,
	        singleSelect: true, 
	        idField: 'eNB_IP',
	        columns: [[
	    		{field:'enbIp',fixed:false,width:350,title:'local IP'},
	    		{field:'enbPort',fixed:false,width:350,title:'local Port'},
	    		{field:'mmeIp',width:350,fixed:false,title:'remote IP'},
	    		{field:'mmePort',fixed:false,width:350, title: 'remote Port'},
	    		{field:'egwConfig_operation',width:70,fixed:true,formatter:eGWConfigLinkFormatter,title:'<%=rb.getString("CaoZuo")%>'},
	    		
	    	]],
	        onBeforeLoad:beforeload_eGWConfig,
	        onLoadSuccess:function(){
	        	
	        }
	    });
	
}

//编辑  一条eNB Config
var eNBEditConigParams = {} ;
var editEnbIdindex;
function editeNBConfig(rowdatas,index){
	$("#basicEnbId").siblings('.prompt').text('');
	$("#basicTac").siblings('.prompt').text('');
	addLinkParams = [];
	editLinkParams = [];
	delLinkParams = [];
	$("#neweNBTitle").text('<%=rb.getString("XiuGaiJiZhan")%>');
	$("#saveNewEnbBtn").css('display','none');
	$("#saveEditEnbBtn").css('display','inline-block'); 
	$("#basicEnbId").attr("disabled","true");
	editEnbIdindex = rowdatas.index;
	editconfigTabNum = index;
	$(".neweNBContainer").animate({right:'20px'});
		 //var linktabobj = {"total":2,"row":[{eNB_IP:"111",eNB_PORT:"27",MME_IP:"192.167.1.1",MME_PORT:"89"},{eNB_IP:"222",eNB_PORT:"89",MME_IP:"192.11.1.1",MME_PORT:"89"}]};
		var params = {};
		params["index"] = rowdatas.index
		params["enbId"] = rowdatas.enb_id;
		params["tac"] = rowdatas.tac;
		params["gwIp"] = linkData.gwIp;
		params["gwPort"] = linkData.gwPort;
		$.post('${ctx}/egw/config/getEnbLinkConfig.action',params,function(data){
			 eNBEditConigParams = data;
			 $("#basicEnbId").val(rowdatas.enb_id);
			 $("#basicTac").val(rowdatas.tac);
			 $("#addLinkConfigTable").datagrid({
			    	data:data.linkList, 
			    	border: true,
			    	fit: true,
			    	queryParams : {
			            timeZone : timeZone
			        },
			    	rownumbers:true,
			    	fitColumns: true,
			        striped: true,
			        singleSelect: true, 
			        idField: 'eNB_IP',
			        columns: [[
						{field:'index',hidden:true,fixed:false,width:350,title:' '},
			    		{field:'enbIp',fixed:false,width:350,title:'local IP'},
			    		{field:'enbPort',fixed:false,width:350,title:'local Port'},
			    		{field:'mmeIp',width:350,fixed:false,title:'remote IP'},
			    		{field:'mmePort',fixed:false,width:350, title: 'remote Port'},
			    		{field:'egwConfig_operation',width:70,fixed:true,formatter:eGWConfigLinkFormatter,title:'<%=rb.getString("CaoZuo")%>'},
			    		
			    	]],
			        onBeforeLoad:beforeload_eGWConfig,
			        onLoadSuccess:function(){
			        	
			        }
			    });
		 
		 },'json')
	
	
}

//添加 一条enbconfig
function addNewEnbSave(){
	var isEdit = true; //判断是否修改的标识
	var params = {};
	params["gwIp"] = linkData["gwIp"];//必传参数无需对比
	params["gwPort"] = linkData["gwPort"];//必传参数无需对比
	                                                 
	params["upLinkS1APToEgwIpOld"] = editConfigData["upLinkS1APToEgwIp"];
	params["upLinkS1APToMMEIpOld"] = editConfigData["upLinkS1APToMMEIp"];													 
	var plmnNewVal = $("#eGWplmn").val();
	var plmnReg = /^\d{5,6}$/
	if(plmnNewVal == ""){
		$(".configConatiner").animate({scrollTop:'30px'});
		$("#eGWplmn").siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDePLMN")%>');
		$("#eGWplmn").siblings('.prompt').css('color','red');
		return;
	}

	if(plmnNewVal != editConfigData.plmn && plmnReg.test(plmnNewVal)){
		 params["plmn"] = plmnNewVal;
		 isEdit = false;
	}else if(plmnNewVal == editConfigData.plmn){
		isEdit = true;
	}else {
		$(".configConatiner").animate({scrollTop:'30px'});
		$("#eGWplmn").siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDePLMN")%>');
		$("#eGWplmn").siblings('.prompt').css('color','red');
		return;
	}
	var levelTwo = $("#eGWMode").combobox('getValue');
	params["levelTwo"] = levelTwo;
	
	//level2 情况下判断 uplink S1AP to egw Ip uplink s1ap to mme ip 不为空
	if(levelTwo == "1" && (editConfigData["upLinkS1APToEgwIp"] == "" && editConfigData["upLinkS1APToMMEIp"] == "")){
		var upLinkToEgwIpVal = $("#eGWUplinkeGWIp").val();
		if(upLinkToEgwIpVal == ""){
			$(".configConatiner").animate({scrollTop:'200px'});
			$("#eGWUplinkeGWIp").siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDeIpDiZhi") %>');
			return;
		}
		if(upLinkToEgwIpVal != editConfigData.upLinkS1APToEgwIp){
			params["upLinkS1APToEgwIp"] = upLinkToEgwIpVal;
			$("#eGWUplinkMMEIp").siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDeIpDiZhi") %>');
			isEdit = false;
		} 
		
		var eGWUplinkMMEIpVal = $("#eGWUplinkMMEIp").val();
		if(eGWUplinkMMEIpVal == ""){
			$(".configConatiner").animate({scrollTop:'200px'});
			return;
		}                                       
		if(eGWUplinkMMEIpVal != editConfigData.upLinkS1APToMMEIp){
			params["upLinkS1APToMMEIp"] = eGWUplinkMMEIpVal;
			isEdit = false;
		} 
	}else if(levelTwo == "0" && (editConfigData["upLinkS1APToEgwIp"] != "" || editConfigData["upLinkS1APToMMEIp"] != "")){
		isEdit = false;
	}
	
	
	//判断slmmetoenbip
	var mmeIpToEnb = $("#eGWMMEIPeNB").siblings('.addIp').find('span');
	var mmeStr = "";
	for(var i = 0;i<mmeIpToEnb.length;i++){
		mmeStr += $(mmeIpToEnb).eq(i).text()+','
	}
	mmeStr = mmeStr.substring(0,mmeStr.length-1)
	if(mmeStr == ""){
		var newmmeArr = [];
	}else{
		var newmmeArr = mmeStr.split(","); //获取到填写的ip作比较是都有变化       
	}
	if(newmmeArr.length == 0){
		$("#eGWMMEIPeNB").siblings('.prompt').text('<%=rb.getString("LinkXinXiBuNengShaoYuYiTiao") %>');
		$(".configConatiner").animate({scrollTop:'260px'});
		return;
	}
	if(newmmeArr.length != editConfigData.s1MmeIpToEnb.length){//先判断数组长度  如果不相等直接传参
		//params["s1MmeIpToEnb"] = newmmeArr;  之前的需求
		params["s1MmeIpToEnb"] = s1MmeIpToEnbADD;
		params["s1MmeIpToEnbDel"] = s1MmeIpToEnbDel;
		params["s1MmePortToEnb"] = $("#eGWMMEPorteNB").val();
		isEdit = false;
	}else{
		if(editConfigData.s1MmeIpToEnb.sort().toString() == newmmeArr.sort().toString()){
			
		}else{
			/* params["s1MmeIpToEnb"] = newmmeArr; */
			params["s1MmeIpToEnb"] = s1MmeIpToEnbADD;
			params["s1MmeIpToEnbDel"] = s1MmeIpToEnbDel;
			params["s1MmePortToEnb"] = $("#eGWMMEPorteNB").val();
			isEdit = false;
		}
	} 
	
	/* var slmmePort = $("#eGWMMEPorteNB").val(); */
	/* if(slmmePort == ""){
		$(".configConatiner").animate({scrollTop:'260px'});
		return;
	} */
	/* if(editConfigData.s1MmePortToEnb != slmmePort){
		params["s1MmePortToEnb"] = slmmePort;
		isEdit = false;
	} */
	
	//判断egw uplink GTPU Ip
	var egwUpLinkGtpuIp = $("#GWUplinkGTPU").siblings('.addIp').find('span');
	var egwUpLinkGtpuIpStr = "";
	for(var i = 0;i<egwUpLinkGtpuIp.length;i++){
		egwUpLinkGtpuIpStr += $(egwUpLinkGtpuIp).eq(i).text()+','
	}
	egwUpLinkGtpuIpStr = egwUpLinkGtpuIpStr.substring(0,egwUpLinkGtpuIpStr.length-1)
	if(egwUpLinkGtpuIpStr == ""){
		var newegwuplinkIpArr = [];
	}else{
		var newegwuplinkIpArr = egwUpLinkGtpuIpStr.split(","); //获取到填写的ip作比较是都有变化       
	}
	
	if(newegwuplinkIpArr.length == 0){
		$("#GWUplinkGTPU").siblings('.prompt').text('<%=rb.getString("BuNengShaoYuYiTiaoIP") %>');
		$(".configConatiner").animate({scrollTop:'400px'});
		return;
	}
	if(newegwuplinkIpArr.length != editConfigData.egwUplinkGtpuIp.length){//先判断数组长度  如果不相等直接传参
		//params["egwUplinkGtpuIp"] = newegwuplinkIpArr;
		params["egwUplinkGtpuIp"] = egwUplinkGtpuIpADD;
		params["egwUplinkGtpuIpDel"] = egwUplinkGtpuIpDel;
		isEdit = false;
	}else{
		if(editConfigData.egwUplinkGtpuIp.sort().toString() == newegwuplinkIpArr.sort().toString()){
			
		}else{
			//params["egwUplinkGtpuIp"] = newegwuplinkIpArr;
			params["egwUplinkGtpuIp"] = egwUplinkGtpuIpADD;
		    params["egwUplinkGtpuIpDel"] = egwUplinkGtpuIpDel;
			isEdit = false;
		}
	}
	
	//判断egw down GTPU Ip
	var egwdownGtpuIp = $("#GWUpDownGTPU").siblings('.addIp').find('span');
	var egwDownGtpuIpStr = "";
	for(var i = 0;i<egwdownGtpuIp.length;i++){
		egwDownGtpuIpStr += $(egwdownGtpuIp).eq(i).text()+','
	}
	egwDownGtpuIpStr = egwDownGtpuIpStr.substring(0,egwDownGtpuIpStr.length-1)
	if(egwDownGtpuIpStr == ""){
		var newegwdownIpArr = [];
	}else{
		var newegwdownIpArr = egwDownGtpuIpStr.split(","); //获取到填写的ip作比较是都有变化      
	}
 
	if(newegwdownIpArr.length == 0){
		$("#GWUpDownGTPU").siblings('.prompt').text('<%=rb.getString("BuNengShaoYuYiTiaoIP") %>');
		$(".configConatiner").animate({scrollTop:'400px'});
		return;
	}
	if(newegwdownIpArr.length != editConfigData.egwDownlinkGtpuIp.length){//先判断数组长度  如果不相等直接传参
		/* params["egwDownlinkGtpuIp"] = newegwdownIpArr; */
		params["egwDownlinkGtpuIp"] = egwDownlinkGtpuIpADD;
		params["egwDownlinkGtpuIpDel"] = egwDownlinkGtpuIpDel;
		isEdit = false;
	}else{
		if(editConfigData.egwDownlinkGtpuIp.sort().toString() == newegwdownIpArr.sort().toString()){
			
		}else{
			/* params["egwDownlinkGtpuIp"] = newegwdownIpArr; */
			params["egwDownlinkGtpuIp"] = egwDownlinkGtpuIpADD;
			params["egwDownlinkGtpuIpDel"] = egwDownlinkGtpuIpDel;
			isEdit = false;
		}
	}
	var slFlexVal = $("#eGWAdvancecFlex").combobox('getValue');
	var logLevelVal = $("#eGWAdvanceLog").combobox('getValue');
	if(slFlexVal == editConfigData.s1FlexEnable){
		
	}else{
		params["s1FlexEnable"] = slFlexVal
		isEdit = false;
	}
	if(logLevelVal == editConfigData.logLevel){
		
	}else{
		params["logLevel"] = logLevelVal;
		isEdit = false;
	}
	params = JSON.stringify(params);
	var param = {};
	param["jsonParams"] = params;
	if(isEdit){ //没有参数修改   收起
		$("#editConfigDiv").slideUp(300);
	}else{
		$.post("${ctx}/egw/config/setBasicConfig.action",param,function(data){
			if(data["success"]){
				if(data["message"] == "0"){
					$("#saveenbConfigBtn").text('<%=rb.getString("XiuGaiHongZhanChengGong")%>')
					$("#saveenbConfigBtn").fadeIn(300,function(){
						setTimeout(function(){
							$("#saveenbConfigBtn").fadeOut(300,function(){
								$("#editConfigDiv").slideUp(300);
							})
						},500)
					})
				}
				if(data["message"] == "1"){
					$("#saveenbConfigBtn").text('<%=rb.getString("XiuGaiHongZhanBuFenChengGong")%>')
					$("#saveenbConfigBtn").fadeIn(300,function(){
						setTimeout(function(){
							$("#saveenbConfigBtn").fadeOut(3000,function(){
								/* $("#editConfigDiv").slideUp(300); */
							})
						},500)
					})
				}
				
			}else{
				showMsg('error_msg',data.message);
			}
		},"json")
	}
	

}


//关闭eNBConfig
function eNBConfigClose(){
	addLinkClose();
	$(".neweNBContainer").animate({right:'-1500px'});
}

//关闭 addlink
function addLinkClose(){
	$("#linkGoback").hide();
	$(".addLinkDiv").slideUp(300);
}
//enbConfig 操作格式化
function eNBConfigFormatter(value, rowData, rowIndex){
	var rowDatas = rowData;
	var index = rowIndex;
	var value = "";
	value += "<div class='operationDiv operation_edit' title='<%=rb.getString("XiuGai") %>' style='width: 20px;height: 20px;margin-left:15px;display:inline-block;cursor:pointer;' onclick='editeNBConfig("+JSON.stringify(rowDatas)+","+index+")'></div>";
	value += "<div class='operationDiv operation_delete' title='<%=rb.getString("ShanChu") %>' style='width: 20px;height: 20px;margin-left:15px;display:inline-block;cursor:pointer;'onclick='deleteeNBConfig("+JSON.stringify(rowDatas)+","+index+")'></div>";
	return value;
}
//enblinkConfig 操作格式化
function eGWConfigLinkFormatter(value,rowData,rowIndex){
	var rowDatas = rowData;
	var index = rowIndex;
	var value = "";
	value += "<div class='operationDiv operation_edit' title='<%=rb.getString("XiuGai") %>' style='width: 20px;height: 20px;margin-left:7px;display:inline-block;cursor:pointer;' onclick='editeNBConfigLink("+JSON.stringify(rowDatas)+","+index+")'></div>";
	value += "<div class='operationDiv operation_delete' title='<%=rb.getString("ShanChu") %>' style='width: 20px;height: 20px;margin-left:7px;display:inline-block;cursor:pointer;'onclick='deleteeNBConfigLink("+JSON.stringify(rowDatas)+","+index+")'></div>";
	return value;
}
//选择egwMode触发事件
function chooseeGWMode(){
	var eGWNodeVal = $("#eGWMode").combobox('getValue');
	if(eGWNodeVal == "1"){
		$(".egwModelevel2").show();
	}else{
		$(".egwModelevel2").hide();
	}
}
//关闭eGW Config
function closeeGWConfigDiv(){
	$("#editConfigDiv").slideUp(300);
}
//校验ip地址
function testTpAddr(e){
	var ipAddrVal = $(e).val();
	var ipReg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
	if(ipReg.test(ipAddrVal)){
		if(removeIp(ipReg,ipAddrVal)){
			$(e).siblings('.prompt').text('');
		}else{
			$(e).siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDeIpDiZhi")%>');
		}
		
	}else{
		$(e).siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDeIpDiZhi")%>');
	}
}
//校验端口范围
function testPort(e){
	var portVal = $(e).val();
	if(portVal<=65535 && portVal>=1){
		$(e).siblings('.prompt').text('');
	}else{
		$(e).siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDeDuanKou")%>');
	}
}

//保存 添加link 数据

function saveAddLink(){
	var row = {};
	var linkDataParams = {};
	/* {eNB_IP:"111",eNB_PORT:"27",MME_IP:"192.167.1.1",MME_PORT:"89"} */
	
	var ipReg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
	var enbIpVal = $("#addEdIteNBIp").val();
	var enbPortVal = $("#addEdIteNBPort").val();
	var mmeIpVal = $("#addEdItMMEIp").val();
	var mmePortVal = $("#addEdItMMEPort").val();
	if(ipReg.test(enbIpVal)){
		if(removeIp(ipReg,enbIpVal)){
			row["enbIp"] = enbIpVal
		}else{
			$("#addEdIteNBIp").siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDeIpDiZhi")%>');
			return;
		}
	}else{
		$("#addEdIteNBIp").siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDeIpDiZhi")%>');
		return;
		
	}
	
	if(ipReg.test(mmeIpVal)){
		if(removeIp(ipReg,mmeIpVal)){
			row["mmeIp"] = mmeIpVal
		}else{
			$("#addEdItMMEIp").siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDeIpDiZhi")%>');
			return;
		}
		
	}else{
		$("#addEdItMMEIp").siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDeIpDiZhi")%>');
		return;
		
	}
	
	if(mmePortVal<=65535 && mmePortVal>=1){
		row["mmePort"] = mmePortVal;
		//$("#addEdItMMEPort).siblings('.prompt').text('');
	}else{
		$("#addEdItMMEPort").siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDeDuanKou")%>');
		return;
		
	} 
	if(enbPortVal<=65535 && enbPortVal>=1){
		row["enbPort"] = enbPortVal;
		//$("#addEdIteNBPort).siblings('.prompt').text('');
	}
	else{
		$("#addEdIteNBPort").siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDeDuanKou")%>');
		return;
	} 
	if(configLinkIndex == -1){
		$("#addLinkConfigTable").datagrid('insertRow',{
			 index:0,
			 row:row
		 });
		addLinkParams.push(row);
		setTimeout(function(){
			$("#addLinkConfigTable").datagrid();
		},0)
		
	}else{
	
		linkDataParams["index"] = editconfigIndex;
		linkDataParams["enbIp"] = $("#addEdIteNBIp").val();
		linkDataParams["enbPort"] = $("#addEdIteNBPort").val();
		linkDataParams["mmeIp"] = $("#addEdItMMEIp").val();
		linkDataParams["mmePort"] = $("#addEdItMMEPort").val();
		if(editconfigIndex == "addLinkSign"){
			delete linkDataParams["index"]; 
			$("#addLinkConfigTable").datagrid('updateRow',{
				 index:configLinkIndex,
				 row:linkDataParams
			 });
			addLinkParams.splice(addLinkParams.length-1-configLinkIndex,1,linkDataParams);
		}
		else{
			$("#addLinkConfigTable").datagrid('updateRow',{
				 index:configLinkIndex,
				 row:linkDataParams
			 });
			editLinkParams.push(linkDataParams);
		}
		
		setTimeout(function(){
			$("#addLinkConfigTable").datagrid();
		},0)
	}
	
	 
	 addLinkClose();
}
//保存 编辑link 数据
/* function saveEditLink(){
	
} */
//编辑link数据
var configLinkIndex = -1;
var editconfigIndex;
function editeNBConfigLink(rowDatas,index){
	$("#linkGoback").show();
	configLinkIndex = index;
	if("index" in rowDatas){
		editconfigIndex = rowDatas.index;
	}else{
		editconfigIndex = "addLinkSign";
	}
	
	$(".addLinkDiv").slideDown(300);
	/* $("#saveEditLink").css("display","block"); */
	/* $("#saveAddLink").css("display","none"); */
	$("#addLinkEnbId").val(rowDatas.index);//标识位
	$("#addEdIteNBIp").val(rowDatas.enbIp);
	$("#addEdIteNBPort").val(rowDatas.enbPort);
	$("#addEdItMMEIp").val(rowDatas.mmeIp);
	$("#addEdItMMEPort").val(rowDatas.mmePort);
	$("#addEdIteNBIp").siblings('.prompt').text('');
	$("#addEdIteNBPort").siblings('.prompt').text('');
	$("#addEdItMMEIp").siblings('.prompt').text('');
	$("#addEdItMMEPort").siblings('.prompt').text('');
}
//添加addLink
function addLink(way){
	$("#linkGoback").show();
	configLinkIndex = -1;
	editconfigIndex = "addLinkSign";
	var linkListNum = $("#addLinkConfigTable").datagrid('getRows').length;
	if(linkListNum >=8){
		showMsg('prompt_msg','<%=rb.getString("LinkXinXiBuNengChaoGuoBaTiao")%>');
		return;
	}
	$(".addLinkDiv").slideDown(300);
	/* $("#saveEditLink").css("display","none");
	$("#saveAddLink").css("display","block"); */
	$("#addEdIteNBIp").val("");
	$("#addEdIteNBIp").siblings('.prompt').text('');
	$("#addEdIteNBPort").val("");
	$("#addEdIteNBPort").siblings('.prompt').text('');
	$("#addEdItMMEIp").val("");
	$("#addEdItMMEIp").siblings('.prompt').text('');
	$("#addEdItMMEPort").val("");
	$("#addEdItMMEPort").siblings('.prompt').text('');
}

//校验basic TAC格式
function testBasicTac(e){
	var bacisisok;
	var tacVal = $("#basicTac").val();
	var testNmuber = /^[0-9]*$/
	var tacValarr = tacVal.split(',');
	for(var i=0;i<tacValarr.length;i++){
		if(testNmuber.test(tacValarr[i])){
			if(tacValarr[i] == 65534){
				$("#basicTac").siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDeTACGeShi")%>');
				bacisisok = false;
				return bacisisok; 
			}
			if(tacValarr[i]<1 || tacValarr[i]>65535){
				$("#basicTac").siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDeTACGeShi")%>');
				bacisisok = false;
				break;
			}else{
				$("#basicTac").siblings('.prompt').text('');
				bacisisok = true
			}
		}else{
			$("#basicTac").siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDeTACGeShi")%>');
			isok = false;
			break;
		}
	}
	
	return bacisisok;
}

//保存添加的 enbconfig
function saveNewEnb(){
	var params = {};
	var enbConFigParams = {};
	var tableParams = {};
	tableParams["gwIp"] = linkData["gwIp"];
	tableParams["gwPort"] = linkData["gwPort"];
	params["gwIp"] = linkData["gwIp"];
	params["gwPort"] = linkData["gwPort"];
	var basiceNBId = $("#basicEnbId").val();
	if(basiceNBId == ""){
		$("#basicEnbId").siblings(".prompt").text('<%=rb.getString("QingShuRuZhengQueDeJiZhanId")%>');
		return;
	}else if(basiceNBId>=1 && basiceNBId<=1048575){
		params["enbId"] = basiceNBId;
	}else {
		$("#basicEnbId").siblings(".prompt").text('<%=rb.getString("QingShuRuZhengQueDeJiZhanId")%>');
		return;
	}
	var basciTac = $("#basicTac").val();
	if(basciTac == ""){
		$("#basicTac").siblings(".prompt").text('<%=rb.getString("QingShuRuZhengQueDeTACGeShi")%>');
		return;
	}
	if(testBasicTac()){
		params['tac'] = basciTac;
	} else{
		return;
	}
	enbConFigParams["enb_id"] = basiceNBId;
	enbConFigParams['tac'] = basciTac;
	var linkListData = $("#addLinkConfigTable").datagrid('getRows');
	if(linkListData.length<1){
		showMsg('prompt_msg','<%=rb.getString("LinkXinXiBuNengShaoYuYiTiao")%>');
		return ;
	}else if(linkListData.length>9){
		showMsg('prompt_msg','<%=rb.getString("LinkXinXiBuNengChaoGuoBaTiao")%>');
		return ;
	}else{
		params["add"] = linkListData;
	}
	params["openType"] = "A";//此参数是为了区分   是新加保存还是修改保存
	params = JSON.stringify(params);
	var param = {};
	param["jsonParams"] = params;
	$.post('${ctx}/egw/config/addEnbLinkConfig.action',param,function(data){
		if(data["success"]){
			var newaddTableData = [];
			$.ajax({
				url:'${ctx}/egw/config/getBasicConfig.action',
				type:'POST',
				async:false,
				data:tableParams,
				dataType:'json',
				success:function(dataTable){
					newaddTableData = dataTable["eNB_Config"];
				}
			})
			
			if(editconfigTabNum != -1){
				$("#addeNBConfigTable").datagrid('updateRow',{
					index:editconfigTabNum,
					row:enbConFigParams
				});
				$("#addeNBConfigTable").datagrid("loadData",newaddTableData);
			}else{
				
				$("#addeNBConfigTable").datagrid('insertRow',{
					index:0,
					row:enbConFigParams
				})
				$("#addeNBConfigTable").datagrid("loadData",newaddTableData);
			}
			
			if(data["message"] == "0"){
				$("#saveEnbsuccessTit").text('<%=rb.getString("XinJianHongZhanChengGong")%>')
				$("#saveEnbsuccessTit").fadeIn(300,function(){
					setTimeout(function(){
						$("#saveEnbsuccessTit").fadeOut(300,function(){
							$(".neweNBContainer").animate({right:'-1500px'});
						})
					},1500)
				})
			}else if(data["message"] == "1"){//部分成功面板也收起
				$("#saveEnbsuccessTit").text('<%=rb.getString("XinJianHongZhanBuFenChengGong")%>')
				$("#saveEnbsuccessTit").fadeIn(300,function(){
					setTimeout(function(){
						$("#saveEnbsuccessTit").fadeOut(300,function(){
							$(".neweNBContainer").animate({right:'-1500px'}); 
						})
					},1500)
				})
			}
			
			/* $(".neweNBContainer").animate({right:'-1500px'}); */
			
		}else{
			showMsg('error_msg',data.message);
		}
	},'json')
}
//保存修改的enbconfig
//eNBEditConigParams  保存原有数据的参数
function saveeditNewEnb(){
	var params = {};
	var editEnbconfigParams = {};
	var tableParams = {};
	tableParams["gwIp"] = linkData["gwIp"];
	tableParams["gwPort"] = linkData["gwPort"];
	params["gwIp"] = linkData["gwIp"];//无需判断 是否修改
	params["gwPort"] = linkData["gwPort"];//无需判断  是否修改
	params["index"] = editEnbIdindex;
	var basiceNBId = $("#basicEnbId").val();
	if(basiceNBId == ""){//enbid不能为空
		$("#basicEnbId").siblings(".prompt").text('<%=rb.getString("QingShuRuZhengQueDeJiZhanId")%>');
		return;
	}
	
	var basciTac = $("#basicTac").val();
	if(basciTac == ""){
		$("#basicTac").siblings(".prompt").text('<%=rb.getString("QingShuRuZhengQueDeTACGeShi")%>');
		return;
	}
	/* else if(basciTac == eNBEditConigParams.tac){
		
	} */
	if(testBasicTac()){
		if(basciTac == eNBEditConigParams.tac){
			
		}else{
			params["enbId"] = basiceNBId;
			params['tac'] = basciTac;
		}
	} else{
		return;
	}
	
	editEnbconfigParams["enbId"] = basiceNBId;
	editEnbconfigParams['tac'] = basciTac;
	//判断linkdata的条数
	var linkListData = $("#addLinkConfigTable").datagrid('getRows');
	if(linkListData.length<1){
		showMsg('prompt_msg','<%=rb.getString("LinkXinXiBuNengShaoYuYiTiao")%>');
		return ;
	}else if(linkListData.length>9){
		showMsg('prompt_msg','<%=rb.getString("LinkXinXiBuNengChaoGuoBaTiao")%>');
		return ;
	}
	//判断是否删除过元素 //当delLinkParams ！=0 有删除的数据
	if(delLinkParams.length != 0){
		params["del"] = delLinkParams;
	}
	//判断是否编辑过元素 
	if(editLinkParams.length != 0){
		params["edit"] = editLinkParams;
	}
	//判断是否添加过元素 
	if(addLinkParams.length != 0){
		params["add"] = addLinkParams;
	}
	params = JSON.stringify(params);
	var param = {};
	param["jsonParams"] = params;
	$.post('${ctx}/egw/config/setEnbLinkConfig.action',param,function(data){
		if(data["success"]){
			//提示修改成功并收起
			var newEditTableData = [];
			$.ajax({
				url:'${ctx}/egw/config/getBasicConfig.action',
				type:'POST',
				async:false,
				data:tableParams,
				dataType:'json',
				success:function(dataTable){
					newEditTableData = dataTable["eNB_Config"];
				}
			})
			$("#addeNBConfigTable").datagrid('updateRow',{
					index:editconfigTabNum,
					row:editEnbconfigParams
				});
			$("#addeNBConfigTable").datagrid("loadData",newEditTableData);
			if(data["message"] == "0"){
				$("#saveEnbsuccessTit").text('<%=rb.getString("XiuGaiHongZhanChengGong")%>');
				$("#saveEnbsuccessTit").fadeIn(300,function(){
				setTimeout(function(){
					$("#saveEnbsuccessTit").fadeOut(300,function(){
						$(".neweNBContainer").animate({right:'-1500px'});
					})
				},1500)
			})
			}else if(data["message"] == "1"){//部分成功面板不收起
				$("#saveEnbsuccessTit").text('<%=rb.getString("XiuGaiHongZhanBuFenChengGong")%>');
				$("#saveEnbsuccessTit").fadeIn(300,function(){
				setTimeout(function(){
					$("#saveEnbsuccessTit").fadeOut(300,function(){
						/* $(".neweNBContainer").animate({right:'-1500px'}); */
					})
				},1500)
			})
			}	
		}else{
			showMsg('error_msg',data.message);
		}
	},"json")
}
//删除enbConfig
function deleteeNBConfig(rowdatas,index){
	var params={};
	params["enbId"] = rowdatas.enb_id;
	params["tac"] = rowdatas.tac;
	params["gwIp"] = linkData.gwIp;
	params["gwPort"] = linkData.gwPort;
	params["index"] = rowdatas.index;
	$.messager.confirm({
		title:TiShi,
		msg:'<%=rb.getString("QueDingShanChuJiZhanPeiZhi")%>',
		fn:function(r){
			if(r){
				$.post('${ctx}/egw/config/delEnbConfig.action',params,function(data){
					if(data["success"]){
						$("#addeNBConfigTable").datagrid("deleteRow",index);
						$("#addeNBConfigTable").datagrid("reload");
					}else{
						showMsg('error_msg',data.message);
					}
				},"json")
			}
		}
	}).addClass("seriousConfirm");
	
}

//删除enbConfigLink
function deleteeNBConfigLink(rowdata,index){
	$.messager.confirm({
		title:TiShi,
		msg:'Are you sure to delete the link config',
		fn:function(r){
			if(r){
				var params = {};
				/* params["index"] = rowdata.index; */
				if("index" in rowdata && rowdata["index"] != "addLinkSign"){
						delLinkParams.push(rowdata.index);
					
				}else{
					addLinkParams.splice(addLinkParams.lenght-1-index,1);
				}
				$("#addLinkConfigTable").datagrid('deleteRow',index);
				$("#addLinkConfigTable").datagrid();
			}
		}
	}).addClass("seriousConfirm");
	
}
//校验plmn
function testPlmn(e){
	var plmnReg = /^\d{5,6}$/
	var plmnVal = $(e).val();
	
	if(plmnVal == ""){
		$(e).siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDePLMN")%>');
		$(e).siblings('.prompt').css('color','red');
		return ;
	}else if(plmnReg.test(plmnVal)){
		$(e).siblings('.prompt').text('<%=rb.getString("QingTianRuWuDaoLiuWeiShuZi")%>');
		$(e).siblings('.prompt').css('color','#9AADB9');
	}else{
		$(e).siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDePLMN")%>');
		$(e).siblings('.prompt').css('color','red');
		return ;
	}
}
function testeNBId(e){
	if($(e).val() == ""){
		$(e).siblings(".prompt").text('<%=rb.getString("QingShuRuZhengQueDeJiZhanId")%>');
	}else if($(e).val()>=0 && $(e).val()<=1048575){
		$(e).siblings(".prompt").text('');
	}else{
		$(e).siblings(".prompt").text('<%=rb.getString("QingShuRuZhengQueDeJiZhanId")%>');
	}
}
/* function beforeaddeNBConfigTable(param){
	param["gwIp"] = paramsegwIp;
	param["gwPort"] = paramsegwPort;
}
function loadFilteraddeNBConfigTable(data){
	return {rows: data.eNB_Config,total: data.eNB_Config.length};
} */
//将不符合的ip剔除
function removeIp(ipReg,test) {
	if(ipReg.test(test)) {
		var testarr = test.split(".");
		for(var i = 0; i < testarr.length; i++) {
			if(testarr[i].length < 3) {
				for(var j = 0; j <= (3 - testarr[i].length); j++) {
					testarr[i] = "0" + testarr[i]
				}
			}
		}
		testStr = testarr.join("");
		testStr = parseInt(testStr);
		if(testarr[0] == "000") { //判断0.0.0.0 - 0.255.255.255
			<%-- $(e).siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDeIpDiZhi")%>'); --%>
			return false;
		} else if(testStr >= 100064000000 && testStr <= 100127255255) {
			<%-- $(e).siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDeIpDiZhi")%>'); --%>
			return false;
		} else if(testStr >= 127000000001 && testStr <= 127255255255) {
			<%-- $(e).siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDeIpDiZhi")%>'); --%>
			return false;
		} else if(testStr >= 169254000000 && testStr <= 169254255255) {
			<%-- $(e).siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDeIpDiZhi")%>'); --%>
			return false;
		} else if(testStr >= 192000000000 && testStr <= 192000000255) {
			<%-- $(e).siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDeIpDiZhi")%>'); --%>
			return false;
		} else if(testStr >= 192000002000 && testStr <= 192000002255) {
			<%-- $(e).siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDeIpDiZhi")%>'); --%>
			return false;
		} else if(testStr >= 192088099000 && testStr <= 192088099255) {
			<%-- $(e).siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDeIpDiZhi")%>'); --%>
			return false;
		} else if(testStr >= 198018000000 && testStr <= 198019255255) {
			<%-- $(e).siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDeIpDiZhi")%>'); --%>
			return false;
		} else if(testStr >= 198051100000 && testStr <= 198051100255) {
			<%-- $(e).siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDeIpDiZhi")%>'); --%>
			return false;
		} else if(testStr >= 203000113000 && testStr <= 203000113255) {
			<%-- $(e).siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDeIpDiZhi")%>'); --%>
			return false;
		} else if(testStr >= 224000000000 && testStr <= 239255255255) {
			<%-- $(e).siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDeIpDiZhi")%>'); --%>
			return false;
		} else if(testStr >= 240000000000 && testStr <= 255255255255) {
			<%-- $(e).siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDeIpDiZhi")%>'); --%>
			return false;
		} else {
			<%-- $(e).siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDeIpDiZhi")%>'); --%>
			return true;
		}
	} else {
		<%-- $(e).siblings('.prompt').text('<%=rb.getString("QingShuRuZhengQueDeIpDiZhi")%>'); --%>
		return false;
	}
}
</script>