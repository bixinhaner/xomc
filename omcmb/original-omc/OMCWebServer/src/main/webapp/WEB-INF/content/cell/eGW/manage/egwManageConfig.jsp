<%@ page import="java.util.Locale"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<script type="text/javascript">
	var ctx = "${ctx}";
</script>

<style>
.egwTit{
	height:40px;
	border-bottom:2px solid #F4FAFD;
	padding-left:20px;
	box-sizing:border-box;
    padding-top: 13px;
}
.egwTit li,.newEGWTit li{
	float:left;
	padding:0 20px;
	text-align:center;
	display:inline-block;
	box-sizing: border-box;
	height:26px;    
	line-height: 16px;
	font-size:16px;
	color:#85A8BF;
	margin-right:20px;
	cursor:default;
}
.omcTitleButton, .omcTitleButtonGroup{
	top:35px;
}
.eGwMgt{
	width:100%;
	height:100%;
	background:white;
}
.eGwMgtHeader{
	padding:20px;
	padding-right:70px;
	position:relative;
	background:white;
	z-index:20;
}
.addeGW{
	position:absolute;
	width:42px;
	height:42px;
	background:#339fd9;
	border-radius:25px;
	top:30px;
	right:70px;
	box-shadow:3px 5px 17px rgba(51,153,204,0.3);
}
.addeGW > span{
	display:inline-block;
	position:absolute;
	border-radius:2px;
	background:white;
}
.heng{
	width:20px;
	height:6px;
	left:11px;
	top:18px;
}
.shu{
	width:6px;
	height:20px;
	top:11px;
	left:18px;
}
.eGWInfo{
	height:100%;
}
.newEGW{
	background:white;
	position:absolute;
	z-index:10;
	width:100%;
	height:100%;
	top:-900px;
	opacity:0;
	
}
.editEGW{
	background:white;
	position:absolute;
	z-index:10;
	width:100%;
	height:100%;
	top:-900px;
	opacity:0;
	
}

.operationTit{
	position:absolute;
	top:85px;
	right:75px;
	font-size:16px;
	color:#2c8cbf;
}
.eGWMainPage{
	height:calc(100% - 44px);
	position:relative;
	overflow:hidden;
}
.addToeGW{
	-webkit-transform:rotate(45deg);
	-moz-transform:rotate(45deg);
	-ms-transform:rotate(45deg);
	-o-transform:rotate(45deg);
	transform:rotate(45deg);
}

.newEGWTit{
	height:34px;
	line-height:50px;
	margin-left:90px;
	margin-top:30px;
}
.newEGWTit li{
	color:#c2c2c2;
}
.newEGWTit li.active{
	color:#151515;
}
.NeweGWTab2{
	display:none;
}
.NeweGWTab1 , .NeweGWTab2{
	height:calc(100% - 85px);
	/* width:calc(100% - 50px); */
	padding-left:130px;
	padding-right:100px;
	overflow:auto;
	margin:0px 0 20px;
}
.secondTitle{
	margin-top:20px;
	margin-bottom:5px;
	height:34px;
	border-bottom:2px solid #f7fafd;
	box-sizing:border-box;
}
.secondTitle li{
	line-height:34px;
	float:left;
	padding:0 30px;
	text-align:center;
	display:inline-block;
	height:34px;
	font-size:14px;
	color:#c2c2c2;
	margin-right:20px;
	cursor:default;
	border-bottom:2px solid #b0e1f6;
	box-sizing:border-box;
	color:#92c8df;
}
.eGWBasicInfoItemDiv{
	/* display:inline-block; 
	width:400px;
	margin:8px 40px 0 0;
	vertical-align:top;	 */
	display: inline-block;
    width: 390px;
    margin: 8px 20px 0 20px;
    vertical-align: top;
}
.eGWBasicInfoItemDivNoMarginRight{
	margin-right:20px;
}
.eGWBasicInfoItemDiv label{
	display:block;
	line-height:25px;
	color:#797979;
}
.eGWBasicInfoItemDiv input{
	width:350px;
}
.eGWBasicInfoItemDiv img{
	margin-left:10px;
}
.eGWBasicInfoItemDiv .prompt{
	display:block;
	line-height:20px;
	height:20px;
	color:red;
}
.LinkInfoInfoItemDiv{
	display:inline-block;
	width:300px;
	margin:8px 40px 0 0;
	vertical-align:top;
}
.LinkInfoInfoItemDiv label{
	display:block;
	line-height:25px;
	color:#797979;
}
.LinkInfoInfoItemDiv input{
	width:310px;
}
.LinkInfoInfoItemDiv img{
	margin-left:10px;
}
.LinkInfoInfoItemDiv .prompt{
	display:block;
	line-height:25px;
	height:25px;
	color:red;
}
.switch {
    width:36px;
    height:14px;
    padding:2px;
    border-radius: 30px;
    -webkit-border-radius:30px;
    -moz-border-radius:30px;
    background-color: #838383;
    position: relative;
    display:inline-block;
    float:left;
}	

.btnn {
    width:14px;
    height:14px;
    -webkit-border-radius:30px;
    -moz-border-radius:30px;
    border-radius:30px;
    background-color: #fff;
    position: absolute;
}
.shuntAndCreditSwitch,.UpLinkInfoSwitch{
	height:45px;
	line-height:45px;
}
.shuntAndCreditSwitch span , .UpLinkInfoSwitch span{
	float:left;
	font-size:14px;
	margin-right:10px;
	color:#797979;
}
.shuntAndCreditSwitch .switch , .UpLinkInfoSwitch .switch{
	margin-top:14px;
	height:15px;
	background:#d7d7d7;
}
.shuntAndCreditSwitch .switch .btnn , .UpLinkInfoSwitch .switch .btnn{
	width:15px;
	height:15px;
	left:2px;
}
.shuntChooseTit li{
	cursor:pointer;
}
.shuntAndCreditCont{
	display:none;
}
.shuntAndCredit{
	margin:40px 0 45px;
}
.functionSwitches{
	margin-bottom:25px;
}
.functionSwitchesCont dl{
	display:inline-block;
	margin-top:20px;
	text-align:left;
	width:150px;
}
.functionSwitchesCont dt{
	color:#797979;
	width:auto;
	display:block;
	margin-bottom:15px;
	text-align:left;
}
.functionSwitchesCont dd{
	height:20px;
	text-align:left;
}
.chooseArrow{
	display:inline-block;
	margin-left:15px;
}
.shuntChoose{
	position:relative;
	margin-bottom:30px;
}
.allShuntGroup{
	height:20px;
	display:none;
}
.shuntChooseItem{
	border:1px solid #d1ecf5;
	width:147px;
	position:absolute;
	top:34px;
	background:white;
	display:none;
	z-index:100;
}
.shuntChooseItem li:nth-of-type(1){
	border-bottom:1px solid #d1ecf5;
}
.shuntChooseItem li{
	height:30px;
	line-height:30px;
	text-align:center;
	cursor:pointer;
}
.shuntChooseItem li:hover{
	background:#e1f2fa;
}
.shuntChooseItem li:active{
	background:#c4e6f5;
}
.eGWBasicInfoTit{
	margin-top:0 !important;
}
.localShuntGroupItem-add .localShuntGroupItem-Edit{
	position:relative;
}

.localShuntNumber{
	width:24px;
	height:24px;
	border:1px solid #bcbcbc;
	color:#797979;
	border-radius:13px;
	position:absolute;
	left:-42px;
	top:33px;
	text-align:center;
	line-height:24px;
}
.newMacroldSetting{
	position:absolute;
	width:900px;
	height:91%;
	background:#FFFFFF;
	box-shadow:2px 3px 16px rgba(158,200,222,0.5);
	border:1px solid #4AB3FF;
	right:-1200px;
	top:18px;
	z-index:200;
	padding:10px 0px 20px 20px;
}
.newMacroldSetting h3{
	color:#7993b6;
	font-size:16px;
	font-weight:normal;
	width:835px;
	padding: 0 20px;
    height: 35px;
    line-height: 30px;
}
.newMacroldSetting h3 div{
	float:left;
}
.newMacroldSetting h3 span{
	float:right;
	cursor:pointer;
	display:block;
	width:20px;
	height:20px;
}
.addLink{
	display:inline-block;
	background:#e1f3f8;
	padding:6px 11px;
	margin-left:20px;
}
.addLink b{
	display:inline-block;
	padding:3px 27px;
	background:white;
	font-weight:normal;
	border:1px solid white;
	cursor:pointer;
}
.addLink b:hover{
	border:1px solid #a1ecfd;
}
.addLinkInfoItem{
	position:absolute;
	top:249px;
	left:20px;
	z-index:400;
	width:742px;
	height:250px;
	border:1px solid #d1ecf5;
	display:none;
	background:white;
	padding-left:50px;
	box-shadow:0px 10px 20px rgba(88,146,176,0.3);
}
.addMacroldTable tr{
	height:46px;
	line-height:46px;
}

.MacroldInfo {height:85%;}
</style>

<div class="omcTitleButton" id="omcTitleButton_add">
	<span class="titleButtonText"><%=rb.getString("TianJia")%></span>
	<span class="circleBg add_circle" onmouseenter="showTipText(this)" onmouseleave="showTipText(this)" onclick="rotateR()">
	</span>
</div>
<div class="omcTitleButton" id="omcTitleButton_close" style="display:none;">
	<span class="titleButtonText"><%=rb.getString("GuanBi")%></span>
	<span class="circleBg close_circle" onmouseenter="showTipText(this)" onmouseleave="showTipText(this)" onclick="rotateR()">
	</span>
</div>

<div class="panelDefault" style="width:100%;overflow-x:hidden;">
	<div class="omcPageTitleDiv omcLogLists">
		<ul class="egwTit">
			<li tabtit="eGWMainPage" class="active" style="border-bottom:2px solid #8DCAF9;box-sizing:border-box;"><%=rb.getString("EGWGuanLi")%></li>
		</ul>
	</div>
	<div class="eGWMainPage ">
		<div class="eGWInfo" style="padding-left:20px;">
			
			<div id="tableGateWayDiv" style="height:100%">
				<table class="easyui-datagrid" id="tableGateWayList" fit="true" data-options="border:false,fitColumns:true,singleSelect:true,toolbar:'#toolbar_tableGateWayList',
	                    rownumbers:true,url:'${ctx}/eGW/egwManage/egwNetConfigPage.action?gwnameip=&TimeZone='+timeZone,pageSize:${pageSize},pageList:${pageList},striped:true,
	                    pagination:true,onBeforeLoad:getParamsBeforeLoad,pagePosition:'bottom',idField:'GW_IP',
	                    onRowContextMenu:''">
					<thead>
					<tr>
						<th data-options="field:'GW_NAME',sortable:true" width="80"><%=rb.getString("EGWMingCheng")%></th>
						<th data-options="field:'GW_IP',sortable:true" width="80"><%=rb.getString("EGWIP")%></th>
						<th data-options="field:'GW_PORT',sortable:true" width="50"><%=rb.getString("EGWDuanKou")%></th>
						<th data-options="field:'operation',formatter: operFormatter" width="80"><%=rb.getString("CaoZuo")%></th>
					</tr>
					</thead>
				</table>
			</div>
		</div>
		<div class="newEGW">
			<ul class="newEGWTit">
				<li onclick="TabsTurn(1,this)" id="egwInfoAddId" class="active"><%=rb.getString("EGWXinXi")%></li>
				<li id="macroid_li" onclick="TabsTurn(2,this)"><%=rb.getString("HongZhanPeiZhi")%></li>
			</ul>
			<div class="NeweGWTab1">
			<!-- <div style="width:1500px;height:1px"></div> -->
				<div class="eGWBasicInfo">
					<ul class="eGWBasicInfoTit secondTitle">
						<li><%=rb.getString("JiBenXinXi")%></li>
					</ul>
					<div class="eGWBasicInfoItemDiv">
						<label for="GateWayName"><%=rb.getString("EGWMingCheng")%></label>
						<input id="GateWayName" type="text" value="" maxlength="32" class="easyui-validatebox border border-box" style="width:360px;" onblur="" />
						<span id="GateWayNameCheckSpan" class="prompt" ></span>
				    </div>
				    <div class="eGWBasicInfoItemDiv">
						<label for="GateWayIP"><%=rb.getString("EGWIP")%></label>
						<!-- onporpertychange="fun(this.id,this.value)" oninput="fun(this.id,this.value)"  -->
						<input id="GateWayIP" type="text" ftype="IP" value="" maxlength="15" class="easyui-validatebox border border-box" style="width:360px;" onblur="checkIpfun(this.id,this.value)"/>
						<span id="GateWayIPCheckSpan" class="prompt" ></span>
				    </div>
				    <div class="eGWBasicInfoItemDiv">
						<label for="GateWayPort"><%=rb.getString("EGWDuanKou")%> (0~65535)</label>
						<input id="GateWayPort" type="text" ftype="Port" value="" maxlength="5" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" style="width:360px;" onblur="checkPort(this.id,this.value)" />
						<span id="GateWayPortCheckSpan" class="prompt" ></span>
				    </div>
				    <div class="eGWBasicInfoItemDiv">
						<label for="PLMN">PLMN (Length 5~6bit)</label>
						<input id="PLMN" type="text" value="" maxlength="6" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" style="width:360px;" onblur="checkPlmn(this.id,this.value)"/>
						<span id="PLMNCheckSpan" class="prompt" ></span>
				    </div>
				    <div class="eGWBasicInfoItemDiv">
						<label for="eNBIp">eNB-Access-IP</label>
						<input id="eNBIp" type="text" value="" ftype="IP" maxlength="15" class="easyui-validatebox border border-box" style="width:360px;" onblur="checkIpfun(this.id,this.value)" />
						<span id="eNBIpCheckSpan" class="prompt" ></span>
				    </div>
				    <div class="eGWBasicInfoItemDiv">
						<label for="eNBPort">eNB-Access-Port (0~65535)</label>
						<input id="eNBPort" type="text" value="" ftype="Port" maxlength="5" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" style="width:360px;" onblur="checkPort(this.id,this.value)" />
						<span id="eNBPortCheckSpan" class="prompt" ></span>
				    </div>
				    <div class="eGWBasicInfoItemDiv">
						<label for="S1UeNBIP">S1U-eNB-IP</label>
						<input id="S1UeNBIP" type="text" value="" ftype="IP" maxlength="15" class="easyui-validatebox border border-box" style="width:360px;" onblur="checkIpfun(this.id,this.value)" />
						<span id="S1UeNBIPCheckSpan" class="prompt" ></span>
				    </div>
				    <div class="eGWBasicInfoItemDiv">
						<label for="S1UMMEIP">S1U-MME-IP</label>
						<input id="S1UMMEIP" type="text" value="" ftype="IP" maxlength="15" class="easyui-validatebox border border-box" style="width:360px;" onblur="checkIpfun(this.id,this.value)" />
						<span id="S1UMMEIPCheckSpan" class="prompt" ></span>
				    </div>
				</div>
				<div class="shuntAndCredit">
					<div class="shuntAndCreditSwitch" style="display:none">
						<span>本地分流及话费服务器开关：</span>
						<div class='switch' style='background-color:#d7d7d7' onclick="openSever(this)">
							<div id="lfpSwitch" isopen='false' class='btnn' style='left:2px;'></div>
						</div>
					</div>
					<div class="shuntAndCreditCont">
						<div class="shuntChoose" style="display:none">
							<ul class="shuntChooseTit secondTitle">
								<li><span id="">分流策略</span></li>
							</ul>
							<div class="verM">
							  <input type="radio" name="shuntaddType" id="addshuntw" value="2" style="margin-left:20px;" onclick="chooseShunt(2)"/>
							  <label for="addshuntw" style="cursor:pointer;">无分流</label>
							  <input type="radio" name="shuntaddType" id="addshuntb" value="1" style="margin-left:40px;" onclick="chooseShunt(1)"/>
							  <label for="addshuntb" style="cursor:pointer;">部分分流</label>
							  <input type="radio" name="shuntaddType" id="addshunta" value="0" style="margin-left:60px;" onclick="chooseShunt(0)"/>
							  <label for="addshunta" style="cursor:pointer;">全部分流</label>
							</div>
							
							<div class="localShuntGroup">
								<div class="defaultlocalShuntGroup-add localShuntGroupItem-add" >
									<div class="localShuntNumber">1</div>
									<div class="eGWBasicInfoItemDiv">
										<label for="gwDNIP">Net-IP</label>
										<input name="gwDNIP" id="gwDNIP" type="text" value="" maxlength="15" class="easyui-validatebox border border-box" style="width:360px;" onblur="" />
										<span name="gwDNIPCheckSpan" id="gwDNIPCheckSpan" class="prompt" ></span>
						   			 </div>
						   			 <div class="eGWBasicInfoItemDiv" style="margin-right:0;">
										<label for="gwDNM">Net-Mask</label>
										<input name="gwDNM" id="gwDNM" type="text" value="" maxlength="15" class="easyui-validatebox border border-box" style="width:360px;" onblur="" />
										<span name="gwDNMCheckSpan" id="gwDNMCheckSpan" class="prompt" ></span>
						   			 </div>
						   			 <div class="addLocalShuntGroup-dis" ></div>
						   			 <div class="addLocalShuntGroup" id="addLocalShuntGroup" onclick="addLocalShuntGroup()"></div>
								</div>
							</div>
							<div class="allShuntGroup">
							</div>
						</div>
						<div class="creditServer">
							<ul class="creditServerTit secondTitle">
								<li><%=rb.getString("HuaDanFuWuQi")%></li>
							</ul>
							<div class="eGWBasicInfoItemDiv">
								<label for="CDR-Generate-IP">CDR-Generate-IP</label>
								<input id="CDR-Generate-IP" type="text" value=""  class="easyui-validatebox border border-box" style="width:360px;" onblur="checkIpfun(this.id,this.value)" />
								<span id="CDR-Generate-IPCheckSpan" class="prompt" ></span>
				   			 </div>
				   			 <div class="eGWBasicInfoItemDiv">
								<label for="CDR-Service-IP">CDR-Service-IP</label>
								<input id="CDR-Service-IP" type="text" value=""  class="easyui-validatebox border border-box" style="width:360px;" onblur="checkIpfun(this.id,this.value)" />
								<span id="CDR-Service-IPCheckSpan" class="prompt" ></span>
				   			 </div>
				   			 <div class="eGWBasicInfoItemDiv">
								<label for="CDR-Number">CDR-Number</label>
								<input id="CDR-Number" type="text" value="" maxlength="4" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" style="width:360px;" onblur="" />
								<span id="CDR-NumberCheckSpan" class="prompt" ></span>
				   			 </div>
						</div>
					</div>		
				</div>
				<div class="functionSwitches" style="display:none">
					<ul class="functionSwitchesTit secondTitle">
						<li><%=rb.getString("GongNengKaiGuan")%></li>
					</ul>
					<div class="functionSwitchesCont">
						<dl>							
							<dt>eNB-DynamicReg</dt>
							<dd>
								<div class='switch' id="gwDynamicreg-Swtich" style='background-color:#d7d7d7' onclick="openFunctions(this)">
									<div id="gwDynamicreg" isopen='false' class='btnn' style='left:1px;'></div>
								</div>
							</dd>
						</dl>
						<dl>							
							<dt>DataRelay</dt>
							<dd>
								<div class='switch' id="gwDatarelay-Swtich" style='background-color:#d7d7d7' onclick="openFunctions(this)">
									<div id="gwDatarelay" isopen='false' class='btnn' style='left:1px;'></div>
								</div>
							</dd>
						</dl>
						<dl>							
							<dt>SigFw</dt>
							<dd>
								<div class='switch' id="gwSigfw-Swtich" style='background-color:#d7d7d7' onclick="openFunctions(this)">
									<div id="gwSigfw" isopen='false' class='btnn' style='left:2px;'></div>
								</div>
							</dd>
						</dl>
						<dl>							
							<dt>UpLinkSelect</dt>
							<dd>
								<div class='switch' id="gwUplinkselect-Swtich" style='background-color:#d7d7d7' onclick="openFunctions(this)">
									<div id="gwUplinkselect" isopen='false' class='btnn' style='left:1px;'></div>
								</div>
							</dd>
						</dl>
						<dl>							
							<dt>eGW-Log</dt>
							<dd>
								<div class='switch' id="gweGWLog-Swtich" style='background-color:#d7d7d7' onclick="openFunctions(this)">
									<div id="gweGWLog" isopen='false' class='btnn' style='left:1px;'></div>
								</div>
							</dd>
						</dl>
						<dl style="width:50px;">							
							<dt>Timer</dt>
							<dd>
								<div class='switch' id="gwTimer-Swtich" style='background-color:#d7d7d7;display:inline;float:left;' onclick="openFunctions(this)">
									<div id="gwTimer" isopen='false' class='btnn' style='left:1px;'></div>
								</div>
							</dd>
						</dl>
						<dl>
							<dt></dt>
							<dd>
								<div id="timers-div" style="float:left;margin-top:-5px;" class="eGWBasicInfoItemDiv">
									<input type="text" value="" maxlength="4" onkeyup="value=value.replace(/[^\d]/g,'')" id="timers" class="easyui-validatebox border border-box" style="width:80px;"/>
									<span id="timersCheckSpan" class="prompt" ></span>
								</div>
							</dd>
						</dl>
					</div>
				</div>
				<p id="submitConfig">
			    	<a href="#" class="linkbutton"  onclick="addGateWaySetUpInfo()"><span><%=rb.getString("BaoCun")%></span></a>
			    </p>
			</div>
			<div class="NeweGWTab2">
				<div class="MacroldInfo">
					<div style="padding:10px 0 20px 0px">
						<div class="queryGroup">
							<input name="" value="" placeholder="<%=rb.getString("HongZhanID")%>" />
							<b onclick="" ></b>
						</div> 
						<a onclick="newMacroldSetting('newAdd')" style="margin-left:20px;" class="linkbutton"><span><%=rb.getString("XinJianHongZhan")%></span></a> 
					</div>
					<div style="height:100%;width:100%;">
						<table class="easyui-datagrid" id="tablemacrocfglist_new" fit="true" data-options="border:false,fitColumns:true,singleSelect:true,
				                    rownumbers:true,url:'${ctx}/eGW/egwManage/egwMacroCfgsList.action?TimeZone='+timeZone,pageSize:${pageSize},pageList:${pageList},striped:true,
				                    pagination:false,onBeforeLoad:getParamsMacrocfgLoad,pagePosition:'bottom',idField:'tac',
				                    onRowContextMenu:''">
							<thead>
							<tr>
								<th data-options="field:'macroId',sortable:true" width="100"><%=rb.getString("HongZhanID")%></th>
								<th data-options="field:'tac',sortable:true" width="100"><%=rb.getString("TAC")%></th>
								<th data-options="field:'operation',formatter: operMacroCfgAddFormatter,sortable:false" width="100"></th>
							</tr>
							</thead>
						</table>
					</div>
				</div>
				<div class="newMacroldSetting">
				<h3><%=rb.getString("XinJianHongZhan")%><span style="" onclick="closeNewMacroldSetting('newAdd')"><span class="egw-del" style="display: inline-block; width:16px; height:16px;"></span></span></h3>
				<div class="eGWBasicInfoItemDiv">
					<label for="New-GatwayName"><%=rb.getString("EGWMingCheng")%></label>
					<input id="New-GatwayName" disabled="true" type="text" value=""  class="easyui-validatebox border border-box" style="width:360px;" onblur="" />
					<span id="New-GatwayNameCheckSpan" class="prompt" ></span>
			 	</div>
			 	<div class="eGWBasicInfoItemDiv">
					<label for="New-GatwayIP"><%=rb.getString("EGWIP")%></label>
					<input id="New-GatwayIP" disabled="true" type="text" value="" maxlength="6" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" style="width:360px;" onblur="checkIpfun(this.id,this.value)" />
					<input type="hidden" id="New-GatwayPort"/>
					<span id="New-GatwayIPCheckSpan" class="prompt" ></span>
			 	</div>
			 	<div class="eGWBasicInfoItemDiv">
					<label for="New-MacroId"><%=rb.getString("HongZhanID")%></label>
					<input id="New-MacroId" type="text" value="" maxlength="16" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" style="width:360px;" onblur="" />
					<input type="hidden" id="New-MacroId-Hidden"/>
					<span id="New-MacroIdCheckSpan" class="prompt" ></span>
			 	</div>
			 	<div class="eGWBasicInfoItemDiv">
					<label for="new-TAC">TAC</label>
					<input id="new-TAC" type="text" value="" maxlength="6" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" style="width:360px;" onblur="" />
					<input type="hidden" id="new-TAC-Hidden"/>
					<span id="new-TACCheckSpan" class="prompt" ></span>
			 	</div>
			 	<div style="position:relative;">
			 		<span><%=rb.getString("LianLu")%></span><span class="addLink" id="addLink_New"><b><%=rb.getString("TianJia")%></b></span>
				 	<div class="addLinkInfoItem">
				 		<div class="UpLinkInfoSwitch" style="display:none">
				 			<span>UpLinkInfo_Switch：</span>
							<div class='switch' id="upLinkEnable-Add-Switch" style='background-color:#d7d7d7' onclick="openSever(this)">
								<div isopen='false' id="upLinkEnable-Add" class='btnn' style='left:2px;'></div>
							</div>
				 		</div>
				 		<br/>
				 		<div class="LinkInfoGroup">
				 			<div class="LinkInfoInfoItemDiv LinkInfoDefault">
								<label >Macro_IP</label>
								<input type="text" value="" id="macroIp-add" maxlength="15" class="easyui-validatebox border border-box" style="width:260px;" onblur="checkIpfun(this.id,this.value)" />
								<img class="addMacroIP" id="macroIpthis" onclick="addMIP(this)"  src="${ctx}/css/images/bi/IP-add.png" style="display:none;"/>
								<span name="macroIp-addCheckSpan" id="macroIp-addCheckSpan" class="prompt" ></span>
						    </div>
						    <div class="LinkInfoInfoItemDiv">
								<label >Macro_Port （0~65535）</label>
								<input  type="text" value=""  maxlength="5" onkeyup="value=value.replace(/[^\d]/g,'')" id="macroPort-add" class="easyui-validatebox border border-box" style="width:260px;"  onblur="checkPort(this.id,this.value)" />
								<span name="macroPort-addCheckSpan" id="macroPort-addCheckSpan" class="prompt" ></span>
						    </div>
						    <!-- 记录原值，修改作对比 -->
						    <input type="hidden" id="macroIp-add-hidden"/>
						    <input type="hidden" id="macroPort-add-hidden"/>
				 		</div>
				 		<div>
				 			<div class="LinkInfoInfoItemDiv LinkInfommeIp">
								<label>MME_Ip</label>
								<input  type="text" value="" maxlength="15" id="mmeIp-add" class="easyui-validatebox border border-box" style="width:260px;" onblur="checkIpfun(this.id,this.value)" />
								<img class="addMacroIP" id="mmeIpthis" onclick="addMIP(this)" src="${ctx}/css/images/bi/IP-add.png" style="display:none;"/>
								<span name="mmeIp-addCheckSpan" id="mmeIp-addCheckSpan" class="prompt" ></span>
						    </div>
						    <div class="LinkInfoInfoItemDiv">
								<label >MME_Port (0~65535)</label>
								<input  type="text" value=""  maxlength="5" onkeyup="value=value.replace(/[^\d]/g,'')" id="mmePort-add" class="easyui-validatebox border border-box" style="width:260px;"  onblur="checkPort(this.id,this.value)" />
								<span name="mmePort-addCheckSpan" id="mmePort-addCheckSpan" class="prompt" ></span>
						    </div>
						    <!-- 记录原值，修改作对比 -->
						    <input type="hidden" id="mmeIp-add-hidden"/>
						    <input type="hidden" id="mmePort-add-hidden"/>
				 		</div>
				 		<input type="hidden" id="rowIndex-add"/>
					    <!-- 新增配置链路添加修改标识 -->
			 			<input type="hidden" id="opertypemme-add"/>
			 			<!-- 新增配置宏站新增修改标识 -->
			 			<input type="hidden" id="operSaveType-add"/>
				 		<p style="position:absolute;bottom:20px;" class="linkbuttonGroup">
							<a onclick="addLinkInfoTo()" class="linkbutton linkbutton_trend"><span><%=rb.getString("BaoCun")%></span></a> 
							<a onclick="cancelAddMarco()" class="linkbutton linkbutton_nowanna"><span><%=rb.getString("GuanBi")%></span></a> 
						</p>
				 		</div>
		 		 	</div>
		 		 	<div style="height:400px;width:850px;">
						<table class="easyui-datagrid" id="addMacroldTable" fit="true" data-options="border:false,fitColumns:true,singleSelect:true,
				                    rownumbers:true,url:'',pageSize:${pageSize},pageList:${pageList},striped:true,
				                    pagination:false,onBeforeLoad:getParamsMMELoad,pagePosition:'bottom',idField:'tac',
				                    onRowContextMenu:''">
							<thead>
							<tr>
								<th data-options="field:'macroIp',sortable:false"  width="100">Macro_IP</th>
								<th data-options="field:'macroPort',sortable:false" >Macro_Port</th>
								<th data-options="field:'mmeIp',sortable:false"  width="100">MME_Ip</th>
								<th data-options="field:'mmePort',sortable:false" >MME_Port</th>
								<th data-options="field:'upLinkEnable',formatter:upLinkEnableFormatter,sortable:false,hidden:true" >UpLinkInfo_Switch</th>
								<th data-options="field:'operation',formatter: operAddMMEFormatter,sortable:false"  width="50"></th>
							</tr>
							</thead>
						</table>
					</div>
					<p style="position:absolute;bottom:20px;">
					  <a onclick="addMacroConfiguration()" style="margin-left:20px;" class="linkbutton"><span><%=rb.getString("BaoCun")%></span></a> 
				    </p>
				</div>
			</div>
		</div>
		
		<!-- 修改eGW信息面板 -->
		<div class="editEGW">
			<input type="hidden" id="editEGWoper"/>
			<ul class="newEGWTit">
				<li onclick="TabsTurnEdit(1,this)" id="egwInfoEditId" class="active"><%=rb.getString("EGWXinXi")%></li>
				<li onclick="TabsTurnEdit(2,this)"><%=rb.getString("HongZhanPeiZhi")%></li>
			</ul>
			<div class="NeweGWTab1">
				<div class="eGWBasicInfo-Edit">
					<ul class="eGWBasicInfoTit secondTitle">
						<li><%=rb.getString("JiBenXinXi")%></li>
					</ul>
					<div class="eGWBasicInfoItemDiv">
						<label for="GateWayName"><%=rb.getString("EGWMingCheng")%></label>
						<input id="GateWayName_Edit" type="text" value="" maxlength="32" class="easyui-validatebox border border-box" style="width:360px;" />
						<span id="GateWayName_EditCheckSpan" class="prompt" ></span>
				    </div>
				    <div class="eGWBasicInfoItemDiv">
						<label for="GateWayIP"><%=rb.getString("EGWIP")%></label>
						<input id="GateWayIP_Edit" type="text" value="" ftype="IP" class="easyui-validatebox border border-box" style="width:360px;" onblur="checkIpfun(this.id,this.value)" disabled="disabled"/>
						<span id="GateWayIP_EditCheckSpan" class="prompt" ></span>
				    </div>
				    <div class="eGWBasicInfoItemDiv">
						<label for="GateWayPort"><%=rb.getString("EGWDuanKou")%> (0~65535)</label>
						<input id="GateWayPort_Edit" type="text" value="" ftype="Port" maxlength="5" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" style="width:360px;" onblur="" disabled="disabled"/>
						<span id="GateWayPort_EditCheckSpan" class="prompt" ></span>
				    </div>
				    <div class="eGWBasicInfoItemDiv">
						<label for="PLMN">PLMN (Length 5~6bit)</label>
						<input id="PLMN_Edit" type="text" value="" maxlength="6" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" style="width:360px;" onblur="checkPlmn(this.id,this.value)"/>
						<span id="PLMN_EditCheckSpan" class="prompt" ></span>
				    </div>
				    <div class="eGWBasicInfoItemDiv">
						<label for="eNBIp">eNB-Access-IP</label>
						<input id="eNBIp_Edit" type="text" value="" ftype="IP" class="easyui-validatebox border border-box" style="width:360px;" onblur="checkIpfun(this.id,this.value)" />
						<span id="eNBIp_EditCheckSpan" class="prompt" ></span>
				    </div>
				    <div class="eGWBasicInfoItemDiv">
						<label for="eNBPort">eNB-Access-Port (0~65535)</label>
						<input id="eNBPort_Edit" type="text" value="" ftype="Port" maxlength="5" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" style="width:360px;" onblur="checkPort(this.id,this.value)"/>
						<span id="eNBPort_EditCheckSpan" class="prompt" ></span>
				    </div>
				    <div class="eGWBasicInfoItemDiv">
						<label for="S1UeNBIP">S1U-eNB-IP</label>
						<input id="S1UeNBIP_Edit" type="text" value="" ftype="IP" class="easyui-validatebox border border-box" style="width:360px;" onblur="checkIpfun(this.id,this.value)" />
						<span id="S1UeNBIP_EditCheckSpan" class="prompt" ></span>
				    </div>
				    <div class="eGWBasicInfoItemDiv">
						<label for="S1UMMEIP">S1U-MME-IP</label>
						<input id="S1UMMEIP_Edit" type="text" value="" ftype="IP" class="easyui-validatebox border border-box" style="width:360px;" onblur="checkIpfun(this.id,this.value)" />
						<span id="S1UMMEIP_EditCheckSpan" class="prompt" ></span>
				    </div>
				</div>
				<div class="shuntAndCredit">
					<div class="shuntAndCreditSwitch" style="display:none">
						<span>本地分流及话费服务器开关：</span>
						<div class='switch' id="lfpSwitch-Edit-Swtich" style='background-color:#d7d7d7' onclick="openSever(this)">
							<div isopen='false' id="lfpSwitch_Edit" class='btnn' style='left:2px;'></div>
						</div>
					</div>
					<div class="shuntAndCreditCont">
						<div class="shuntChoose" style="display:none">
							<ul class="shuntChooseTit secondTitle">
								<li><span id="">分流策略</span></li>
							</ul>
							<div class="verM">
							  <input type="radio" name="shuntType" id="shuntw" value="2" style="margin-left:20px;" onclick="chooseShuntEdit(2)"/>
							  <label for="shuntw" style="cursor:pointer;">无分流</label>
							  <input type="radio" name="shuntType" id="shuntb" value="1" style="margin-left:40px;" onclick="chooseShuntEdit(1)"/>
							  <label for="shuntb" style="cursor:pointer;">部分分流</label>
							  <input type="radio" name="shuntType" id="shunta" value="0" style="margin-left:60px;" onclick="chooseShuntEdit(0)"/>
							  <label for="shunta" style="cursor:pointer;">全部分流</label>
							</div>
							
							<div class="localShuntGroup_Edit">
								<div class="defaultlocalShuntGroup-Edit localShuntGroupItem-Edit" >
									<div class="localShuntNumber">1</div>
									<div class="eGWBasicInfoItemDiv">
										<label for="gwDNIP">Net-IP</label>
										<input name="gwDNIP_Edit" id="gwDNIP_Edit" type="text" value="" maxlength="15" class="easyui-validatebox border border-box" style="width:360px;" onblur="" />
										<input name="gwDNIP_Edit_Hidden" id="gwDNIP_Edit_Hidden" type="hidden" />
										<span name="gwDNIP_EditCheckSpan" id="gwDNIP_EditCheckSpan" class="prompt" ></span>
						   			 </div>
						   			 <div class="eGWBasicInfoItemDiv" style="margin-right:0;">
										<label for="gwDNM">Net-Mask</label>
										<input name="gwDNM_Edit" id="gwDNM_Edit"  type="text" value="" maxlength="15" class="easyui-validatebox border border-box" style="width:360px;" onblur="" />
										<input name="gwDNM_Edit_Hidden" id="gwDNM_Edit_Hidden" type="hidden" />
										<span name="gwDNM_EditCheckSpan" id="gwDNM_EditCheckSpan" class="prompt" ></span>
						   			 </div>
						   			 <div class="addLocalShuntGroup-dis" ></div>
						   			 <div class="addLocalShuntGroup" id="addLocalShuntGroup" onclick="addLocalShuntGroup_Edit()"></div>
						   			 
								</div>
							</div>
							<div class="allShuntGroup">
							</div>
						</div>
						<div class="creditServerEdit">
							<ul class="creditServerTit secondTitle">
								<li><%=rb.getString("HuaDanFuWuQi")%></li>
							</ul>
							<div class="eGWBasicInfoItemDiv">
								<label for="CDR-Generate-IP">CDR-Generate-IP</label>
								<input id="CDR-Generate-IP-Edit" type="text" value="" ftype="IP" class="easyui-validatebox border border-box" style="width:360px;" onblur="checkIpfun(this.id,this.value)" />
								<span id="CDR-Generate-IP-EditCheckSpan" class="prompt" ></span>
				   			 </div>
				   			 <div class="eGWBasicInfoItemDiv">
								<label for="CDR-Service-IP">CDR-Service-IP</label>
								<input id="CDR-Service-IP-Edit" type="text" value="" ftype="IP" class="easyui-validatebox border border-box" style="width:360px;" onblur="checkIpfun(this.id,this.value)" />
								<span id="CDR-Service-IP-EditCheckSpan" class="prompt" ></span>
				   			 </div>
				   			 <div class="eGWBasicInfoItemDiv">
								<label for="CDR-Number">CDR-Number</label>
								<input id="CDR-Number-Edit" type="text" value="" maxlength="4" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" style="width:360px;" onblur="" />
								<span id="CDR-Number-EditCheckSpan" class="prompt" ></span>
				   			 </div>
						</div>
					</div>		
				</div>
				<div class="functionSwitches" style="display:none;">
					<ul class="functionSwitchesTit secondTitle">
						<li><%=rb.getString("GongNengKaiGuan")%></li>
					</ul>
					<div class="functionSwitchesCont">
						<dl>
							<dt>eNB-DynamicReg</dt>
							<dd>
								<div class='switch' id="gwDynamicreg-Edit-Swtich" style='background-color:#d7d7d7' onclick="openFunctions(this)">
									<div isopen='false' id="gwDynamicreg-Edit" class='btnn' style='left:1px;'></div>
								</div>
							</dd>
						</dl>
						<dl>
							<dt>DataRelay</dt>
							<dd>
								<div class='switch' id="gwDatarelay-Edit-Swtich" style='background-color:#d7d7d7' onclick="openFunctions(this)">
									<div isopen='false' id="gwDatarelay-Edit" class='btnn' style='left:1px;'></div>
								</div>
							</dd>
						</dl>
						<dl>
							<dt>SigFw</dt>
							<dd>
								<div class='switch' id="gwSigfw-Edit-Swtich" style='background-color:#d7d7d7' onclick="openFunctions(this)">
									<div isopen='false' id="gwSigfw-Edit" class='btnn' style='left:2px;'></div>
								</div>
							</dd>
						</dl>
						<dl>
							<dt>UpLinkSelect</dt>
							<dd>
								<div class='switch' id="gwUplinkselect-Edit-Swtich" style='background-color:#d7d7d7' onclick="openFunctions(this)">
									<div isopen='false' id="gwUplinkselect-Edit" class='btnn' style='left:1px;'></div>
								</div>
							</dd>
						</dl>
						<dl>							
							<dt>eGW-Log</dt>
							<dd>
								<div class='switch' id="gweGWLog-Edit-Swtich" style='background-color:#d7d7d7' onclick="openFunctions(this)">
									<div isopen='false' id="gweGWLog-Edit" class='btnn' style='left:1px;'></div>
								</div>
							</dd>
						</dl>
						<dl style="width:50px;">							
							<dt>Timer</dt>
							<dd>
								<div class='switch' id="gwTimer-Edit-Swtich" style='background-color:#d7d7d7;' onclick="openFunctions(this)">
									<div isopen='false' id="gwTimer-Edit" class='btnn' style='left:1px;'></div>
								</div>
							</dd>
						</dl>
						<dl>
							<dt></dt>
							<dd>
								<div id="timers-edit-div" style="float:left;margin-top:-5px;" class="eGWBasicInfoItemDiv">
									<input type="text" value="" maxlength="4" onkeyup="value=value.replace(/[^\d]/g,'')" id="timers-edit" class="easyui-validatebox border border-box" style="width:80px;">
									<span id="timers-editCheckSpan" class="prompt" ></span>
								</div>
							</dd>
						</dl>
					</div>
				</div>
				<p id="submitConfig">
			    	<a href="#" class="linkbutton" onclick="editGateWaySetUpInfo()"><span><%=rb.getString("BaoCun")%></span></a>
			    </p>
			</div>
			<div class="NeweGWTab2">
				<div class="MacroldInfo">
					<div style="padding:10px 0 20px 0px">
						<div class="queryGroup">
							<input name="query-macroid" id="query-macroid" value="" placeholder="<%=rb.getString("HongZhanID")%>" />
							<b onclick="getMacroidInfo()"></b>
						</div> 
						<a onclick="newMacroldSetting('editAdd')" class="linkbutton"><span><%=rb.getString("XinJianHongZhan")%></span></a> 
					</div>
					<div style="height:100%;width:100%;">
						<table class="easyui-datagrid" id="tablemacrocfglist" fit="true" data-options="border:false,fitColumns:true,singleSelect:true,
				                    rownumbers:true,url:'${ctx}/eGW/egwManage/egwMacroCfgsList.action?TimeZone='+timeZone,pageSize:${pageSize},pageList:${pageList},striped:true,
				                    pagination:false,onBeforeLoad:getParamsMacrocfgLoad,pagePosition:'bottom',idField:'tac',
				                    onRowContextMenu:''">
							<thead>
							<tr>
								<th data-options="field:'macroId',sortable:true" width="100"><%=rb.getString("HongZhanID")%></th>
								<th data-options="field:'tac',sortable:true" width="100">TAC</th>
								<th data-options="field:'operation',formatter: operMacroCfgFormatter,sortable:true" width="100"><%=rb.getString("CaoZuo")%></th>
							</tr>
							</thead>
						</table>
					</div>
				</div>
				<div class="newMacroldSetting" >
				<div class="easyui-layout" data-options="border:false,fit:true">
					<div region="north" data-options="border:false" style="width:880px;padding: 10px 0px;height: 260px">
						<h3>
							<div id="macrotitle"></div>
							<span onclick="closeNewMacroldSetting()" class="titleIcon_close iconSize">
							</span>
						</h3>
						<div class="eGWBasicInfoItemDiv">
							<label for="New-GatwayName"><%=rb.getString("EGWMingCheng")%></label>
							<input id="Edit-GatwayName" disabled="true" type="text" value=""  class="easyui-validatebox border border-box" style="width:360px;" onblur="" />
							<span id="New-GatwayNameCheckSpan" class="prompt" ></span>
				 		</div>
					 	<div class="eGWBasicInfoItemDiv">
							<label for="New-GatwayIP"><%=rb.getString("EGWIP")%></label>
							<input id="Edit-GatwayIP" disabled="true" type="text" value=""  class="easyui-validatebox border border-box" style="width:360px;" onblur="checkIpfun(this.id,this.value)"/>
							<input id="Edit-GatwayPort" type="hidden" />
							<span id="New-GatwayIPCheckSpan" class="prompt" ></span>
					 	</div>
					 	<div class="eGWBasicInfoItemDiv">
							<label for="New-MacroId"><%=rb.getString("HongZhanID")%></label>
							<input id="Edit-MacroId" type="text" value="" maxlength="16" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" style="width:360px;" onblur=""/>
							<input type="hidden" id="Edit-MacroId-Hidden"/>
							<span id="Edit-MacroIdCheckSpan" class="prompt" ></span>
					 	</div>
					 	<div class="eGWBasicInfoItemDiv">
							<label for="new-TAC">TAC</label>
							<input id="Edit-TAC" type="text" value="" maxlength="6" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" style="width:360px;" onblur="" />
							<input type="hidden" id="Edit-TAC-Hidden"/>
							<span id="Edit-TACCheckSpan" class="prompt" ></span>
					 	</div>
				 		<div style="position:relative;margin-left:20px;margin-top:10px;">
				 			<span><%=rb.getString("LianLu")%></span><%-- <span class="addLink" id="addLink_Edit"><b><%=rb.getString("TianJia")%></b></span> --%>
				 			<a id="addLink_Edit" class="addLink linkbutton"><span><%=rb.getString("TianJia")%></span></a>
					 		
			 		 	</div>
			 		 	</div>
			 		 	<div region="center"  data-options="border:false" style="width:850px;padding-left:20px;padding-bottom:20px;margin-top:10px;">
							<table class="easyui-datagrid" id="tablemmelist" fit="true" data-options="border:false,fitColumns:true,singleSelect:true,
					                    rownumbers:true,url:'',pageSize:${pageSize},pageList:${pageList},striped:true,
					                    pagination:false,onBeforeLoad:getParamsMMELoad,pagePosition:'bottom',idField:'tac',
					                    onRowContextMenu:''">
								<thead>
								<tr>
									<th data-options="field:'macroIp',sortable:false"  width="100">Macro_IP</th>
									<th data-options="field:'macroPort',sortable:false" >Macro_Port</th>
									<th data-options="field:'mmeIp',sortable:false"  width="100">MME_Ip</th>
									<th data-options="field:'mmePort',sortable:false" >MME_Port</th>
									<th data-options="field:'upLinkEnable',formatter:upLinkEnableFormatter,sortable:false,hidden:true" >UpLinkInfo_Switch</th>
									<th data-options="field:'operation',formatter: operMMEFormatter,sortable:false"  width="50"><%=rb.getString("CaoZuo")%></th>
								</tr>
								</thead>
							</table>
						</div>
						<div region="south"  data-options="border:false" style="height:40px;margin-top:10px;">
							  <a onclick="editMacroConfiguration()" style="margin-left:20px;" class="linkbutton"><span><%=rb.getString("BaoCun")%></span></a> 
					    </div>
					    <div class="addLinkInfoItem linkInfoItemDivEdit">
					 			<div class="UpLinkInfoSwitch" style="display:none;">
					 				<span>UpLinkInfo_Switch：</span>
									<div class='switch' id="upLinkEnable-Edit-Switch" style='background-color:#d7d7d7' onclick="openSever(this)">
										<div isopen='false' id="upLinkEnable-Edit" class='btnn' style='left:2px;'></div>
									</div>
					 			</div>
					 			<br/>
						 		<div class="LinkInfoGroup">
						 			<div class="LinkInfoInfoItemDiv LinkInfoDefault-Edit">
										<label >Macro_IP</label>
										<input type="text" value="" maxlength="15" id="macroIp-edit" class="easyui-validatebox border border-box" style="width:260px;" onblur="checkIpfun(this.id,this.value)" />
										<img class="addMacroIP-Edit" id="macroIpthis-edit" onclick="addMIP_Edit(this)"  src="${ctx}/css/images/bi/IP-add.png" style="display:none;"/>
										<span id="macroIp-editCheckSpan" name="macroIp-editCheckSpan" class="prompt" ></span>
								    </div>
								    <div class="LinkInfoInfoItemDiv">
										<label >Macro_Port (0~65535)</label>
										<input  type="text" value="" maxlength="5" onkeyup="value=value.replace(/[^\d]/g,'')" id="macroPort-edit"  class="easyui-validatebox border border-box" style="width:260px;"  onblur="checkPort(this.id,this.value)" />
										<span id="macroPort-editCheckSpan" name="macroPort-editCheckSpan" class="prompt" ></span>
								    </div>
								    <!-- 记录原值，修改作对比 -->
								    <input type="hidden" id="macroIp-edit-hidden"/>
								    <input type="hidden" id="macroPort-edit-hidden"/>
						 		</div>
					 			<div>
						 			<div class="LinkInfoInfoItemDiv LinkInfommeIp-Edit">
										<label>MME_Ip</label>
										<input  type="text" value="" maxlength="15" id="mmeIp-edit" class="easyui-validatebox border border-box" style="width:260px;" onblur="checkIpfun(this.id,this.value)" />
										<img class="addMacroIP-Edit" id="mmeIpthis-edit" onclick="addMIP_Edit(this)" src="${ctx}/css/images/bi/IP-add.png" style="display:none;"/>
										<span id="mmeIp-editCheckSpan" name="mmeIp-editCheckSpan" class="prompt" ></span>
								    </div>
								    <div class="LinkInfoInfoItemDiv">
										<label >MME_Port (0~65535)</label>
										<input  type="text" value="" maxlength="5" onkeyup="value=value.replace(/[^\d]/g,'')" id="mmePort-edit" class="easyui-validatebox border border-box" style="width:260px;"  onblur="checkPort(this.id,this.value)" />
										<span id="mmePort-editCheckSpan" name="mmePort-editCheckSpan" class="prompt" ></span>
								    </div>
									<!-- 记录原值 ，修改作对比-->
									<input type="hidden" id="mmeIp-edit-hidden"/>
									<input type="hidden" id="mmePort-edit-hidden"/>
					 			</div>
						 		<input type="hidden" id="rowIndex"/>
						 		<!-- 链路新增修改标识 -->
						 		<input type="hidden" id="opertypemme"/>
						 		<!-- 宏站新增修改标识 -->
						 		<input type="hidden" id="operSaveType"/>
						 		<p style="position:absolute;bottom:20px;" class="linkbuttonGroup">
									<a onclick="editLinkInfo()" class="linkbutton linkbutton_trend"><span><%=rb.getString("BaoCun")%></span></a> 
									<a onclick="cancelAddMarco()" class="linkbutton linkbutton_nowanna"><span><%=rb.getString("GuanBi")%></span></a> 
								</p>
					 		</div>
				</div>
				</div>
			</div>
		</div>
	</div>
</div>
<div id="toolbar_tableGateWayList" class="omcTableTool">
        <div class="queryGroup">
          	<input name="gwnameip" id="gwnameip" value=""  placeholder="<%=rb.getString("EGWChaXunTiShi")%>">
			<b onclick="egwNetConfigPage()"></b>
        </div>
</div> 							
<%-- 窗口-进度条 --%>
<div id="winLoadingPro1" title="<%=rb.getString("QinQiuJinDu")%>" class="easyui-window" style="background:url(${ctx}/images/egw_loading_v1.gif)no-repeat;"
		data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:335,height:200,resizable:false,closable:false">
   	<div>
   		<span style="margin-left:150px;margin-top:100px;float:left;"><%=rb.getString("ShuJuQingQiu")%></span>
   	</div>
    <!--<img src="${ctx}/images/egw_loading_v1.gif"/>
    <img src="${ctx}/images/set-loading.gif"/> 
    <img src="${ctx}/skin/BaiCells/images/global/loading.gif" style="margin-left:10px;margin-top:10px;float:left;"/>
    <div style="margin-left:5px;float:right;margin-right:10px;">
    	<img src="${ctx}/images/egw_loading_v1.gif"/><br/>
    	<span style="margin-left:0px;">数据请求中，请稍等……</span>
    </div>-->
</div>

<%-- 窗口-进度条 --%>
<div id="winLoadingPro" title="<%=rb.getString("QinQiuJinDu")%>" class="easyui-window"
		data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:400,height:200,resizable:false,closable:false">
    <span class="loading-gif" style="margin-left:10px;margin-top:10px;"></span>
    <span style="margin-left:75px;"><%=rb.getString("ShuJuQingQiu")%></span>
</div>

<%-- 窗口-NEW ADD EGW--%>
<div id="winAddEGWPro" title="<%=rb.getString("TianJiaYiYouWangGuan")%>" class="easyui-window"
		data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:485,height:380,resizable:false,closable:false">
	<div style="margin-top:10px;margin-left:30px;">
		<ul>
			<li>
				<!-- <span style="color:red;">该网关记录不存在，确认设备端是否已配置，已配置请在此处添加.</span> -->
			</li>
            <li>
                <div class="eGWBasicInfoItemDiv" style="width:300px;">
					<label for="GateWayIP"><%=rb.getString("EGWIP")%></label>
					<input id="GateWayIP-Add" type="text" value="" class="easyui-validatebox border border-box" style="width:260px;" onblur="checkIpfun(this.id,this.value)" />
					<span id="GateWayIP-AddCheckSpan" class="prompt" ></span>
			    </div>
            </li>
            <li>
                <div class="eGWBasicInfoItemDiv" style="width:300px;">
					<label for="GateWayName"><%=rb.getString("EGWMingCheng")%></label>
					<input id="GateWayName-Add" type="text" value="" maxlength="32" class="easyui-validatebox border border-box" style="width:260px;" />
					<span id="GateWayName-AddCheckSpan" class="prompt" ></span>
			    </div>
            </li>
            <li>
                <div class="eGWBasicInfoItemDiv" style="width:300px;">
					<label for="GateWayPort"><%=rb.getString("EGWDuanKou")%></label>
					<input id="GateWayPort-Add" type="text" value="" maxlength="5" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" style="width:260px;" onblur="checkPort(this.id,this.value)" />
					<span id="GateWayPort-AddCheckSpan" class="prompt" ></span>
			    </div>
            </li>
        </ul>
    </div>
    <div style="position:absolute;bottom:20px;margin-left:20px;" class="linkbuttonGroup" >
    	<a href="#" class="linkbutton" onclick="addGateWay()"><span><%=rb.getString("BaoCun")%></span></a>
        <a href="#" class="linkbutton" onclick="$('#winAddEGWPro').window('close')"><span><%=rb.getString("GuanBi")%></span></a>   	
    </div>
</div>

<script type="text/javascript">

	$(function() {
		
		$(document).click(function() {
    		
    			$(".shuntChooseItem").slideUp(300); 
    			$(".chooseArrow").removeClass('egw-arrow-up').addClass('egw-arrow-down');
    	});	
		$("#timers-edit-div,#timers-div").hide();
		//网关表格高度计算
		//$("#tableGateWayDiv").css("height",($(document.body).height()-210)+"px");
	});
	
	//查询网管信息
	function egwNetConfigPage(){
		var gwnameip = $.trim($("#gwnameip").val());
		if(getIseGWIp(gwnameip)&&gwnameip!=""&&isValidIP(gwnameip)){
			$("#GateWayName-Add,#GateWayIP-Add,#GateWayPort-Add").val("");
			$("#GateWayIP-Add").val(gwnameip).attr("disabled","disabled");
			$("#winAddEGWPro").window("open");
		}else{
			$("#tableGateWayList").datagrid({
		    	   url:'${ctx}/eGW/egwManage/egwNetConfigPage.action?TimeZone='+timeZone,
		    	   queryParams:{
		    		   gwnameip : gwnameip
		    	   }
		    });
		}
	}
	
	function addEGWInfo(){
		$("#GateWayName-Add,#GateWayIP-Add,#GateWayPort-Add").val("");
		$("#winAddEGWPro").window("open");
	}
	
	//单独添加网关，对于基站已经配置后
	function addGateWay(){
		$(".prompt").text("");
		var GateWayName = $.trim($("#GateWayName-Add").val());
		if(GateWayName==""){
			$("#GateWayName-Add").focus().select();
			$("#GateWayName-AddCheckSpan").text("<%=rb.getString("BuNengWeiKong")%>");
			return false;
		}		
		var GateWayIP = $.trim($("#GateWayIP-Add").val());
		if(GateWayIP==""){
			$("#GateWayIP-Add").focus().select();
			$("#GateWayIP-AddCheckSpan").text("<%=rb.getString("BuNengWeiKong")%>");
			return false;
		}
		if(!isValidIP(GateWayIP)){
			$("#GateWayIP-Add").focus().select();
			$("#GateWayIP-AddCheckSpan").text("<%=rb.getString("IPGeShiBuDui")%>");
			return false;
		}	
		var GateWayPort = $.trim($("#GateWayPort-Add").val());
		if(GateWayPort==""){
			$("#GateWayPort-Add").focus().select();
			$("#GateWayPort-AddCheckSpan").text("<%=rb.getString("BuNengWeiKong")%>");
			return false;
		}
		if(!checkPort("GateWayPort-Add",GateWayPort)){
			return false;
		}
		$("#winLoadingPro").window("open");
		var params = {gwname:GateWayName,gwip:GateWayIP,gwport:GateWayPort};
		$.ajax({
			type: "post",
			url: "${ctx}/eGW/egwManage/addGateWay.action?TimeZone="+timeZone,
			data: params,
			async: true,
			dataType:"json",
			success: function(data) {
				$("#winLoadingPro").window("close");
				$("#winAddEGWPro").window("close");
				if(data["message"]=="0"){
					$("#tableGateWayList").datagrid("reload");	
				}else if(data["message"]=="1"){
					$("#GateWayIP-Add").focus().select();
					$("#GateWayIP-AddCheckSpan").text("<%=rb.getString("WangGuanYiCunZaiQingChongXinShuRu")%>");
				}else{
					$.messager.alert(TiShi, "<%=rb.getString("TianJiaShiBaiLianJieYiChang")%>");
				}
				$("#tableGateWayList").datagrid({
			    	   url:'${ctx}/eGW/egwManage/egwNetConfigPage.action?TimeZone='+timeZone,
			    	   queryParams:{
			    		   gwnameip : "",
			    		   orderdata : GateWayIP
			    	   }
			    });
			},
			error:function(xmlhttprequest,textstatus,errorThrown){
				$("#winAddEGWPro").window("close");
				$("#winLoadingPro").window("close");
				if(textstatus=="timeout"){
					$.messager.alert(TiShi, "<%=rb.getString("QingQiuChaoShi")%>");
				}else{
					$.messager.alert(TiShi, "<%=rb.getString("QingQiuYiChang")%>");
				}
			}
		});
	}
	//wjw检测添加eGB
	 var eGBwidth = $(document).width();
	 
	 var sel = $(".eGWBasicInfoItemDiv:not('.newMacroldSetting .eGWBasicInfoItemDiv')");
		if(eGBwidth<=1280){
			sel.css("width","180px");
			sel.css("marginright","15px");
			sel.find("input").css("width","140px");
		}
		else if(eGBwidth>1280 && eGBwidth<=1359){
			sel.css("width","180px");
			sel.css("marginright","15px");
			sel.find("input").css("width","140px");
		}else if(eGBwidth>1359 && eGBwidth<=1614){
			sel.css("width","215px");
			sel.css("marginright","20px");
			sel.find("input").css("width","180px");
		}else if(eGBwidth>1614 && eGBwidth<=1704){
			sel.css("width","300px");
			sel.css("marginright","22px");
			sel.find("input").css("width","210px");
		}
		else if(eGBwidth>1704 && eGBwidth<=1765){
			sel.css("width","330px");
			sel.css("marginright","25px");
			sel.find("input").css("width","230px");
		}
		else if(eGBwidth>1765 && eGBwidth<=1914 ){
			sel.css("width","350px");
			sel.css("marginright","30px");
			sel.find("input").css("width","230px");
		}
		else{
			sel.css("width","400px");
			sel.css("marginright","40px");
			sel.find("input").css("width","360px");
		}
	
	 $(window).resize(function(){
		var sel = $(".eGWBasicInfoItemDiv:not('.newMacroldSetting .eGWBasicInfoItemDiv')");
		var nowwidth = $(document).width();
		if(nowwidth<=1280){
			sel.css("width","180px");
			sel.css("marginright","15px");
			sel.find("input").css("width","140px");
		}
		else if(nowwidth>1280 && nowwidth<=1359){
			sel.css("width","180px");
			sel.css("marginright","15px");
			sel.find("input").css("width","140px");
		}else if(nowwidth>1359 && nowwidth<=1614){
			sel.css("width","220px");
			sel.css("marginright","20px");
			sel.find("input").css("width","180px");
		}else if(nowwidth>1614 && nowwidth<=1704){
			sel.css("width","300px");
			sel.css("marginright","22px");
			sel.find("input").css("width","210px");
		}
		else if(nowwidth>1704 && nowwidth<=1765){
			sel.css("width","330px");
			sel.css("marginright","25px");
			sel.find("input").css("width","230px");
		}
		else if(nowwidth>1765 && nowwidth<=1914 ){
			sel.css("width","350px");
			sel.css("marginright","30px");
			sel.find("input").css("width","230px");
		}
		else{
			sel.css("width","400px");
			sel.css("marginright","40px");
			sel.find("input").css("width","360px");
		}
	}) 
	
	
	//添加/关闭eGW 按钮点击事件 
	function rotateR(){
		chooseShunt(2);
		emptyAll();
		var rBottom = $(".newEGW").position().top;
		TabsTurn(1,$("#egwInfoAddId"));
		if($("#editEGWoper").val()=="open"){
			rBottom=0;
			$("#editEGWoper").val("close");
		}
		if(rBottom <-800){
			//$(".addeGW").children().addClass("addToeGW");
			$("#omcTitleButton_add").hide();
			$("#omcTitleButton_close").show();
			$(".newEGW").animate({top:"0",opacity:"1"});
			$(".egwTit li").text("<%=rb.getString("XinJianEGW")%>");
			$(".operationTit").text("<%=rb.getString("GuanBi")%>");
		}else{
			//$(".addeGW").children().removeClass("addToeGW");
			$("#omcTitleButton_close").hide();
			$("#omcTitleButton_add").show();
			$(".newEGW").animate({top:"-900px",opacity:"0"});
			$(".editEGW").animate({top:"-900px",opacity:"0"});
			$(".egwTit li").text("<%=rb.getString("EGWGuanLi")%>");
			$(".operationTit").text("<%=rb.getString("TianJia")%>");
		}
	}
	
	function rotateR_Edit(){
		var rBottom = $(".editEGW").position().top;
		TabsTurnEdit(1,$("#egwInfoEditId"));
		if(rBottom <-800){
			$("#editEGWoper").val("open");
			$("#omcTitleButton_add").hide();
			$("#omcTitleButton_close").show();
			$(".editEGW").animate({top:"0",opacity:"1"})
			$(".egwTit li").text("<%=rb.getString("XiuGaiEGW")%>");
			$(".operationTit_Edit").text("<%=rb.getString("GuanBi")%>");			
		}else{
			$("#omcTitleButton_close").hide();
			$("#omcTitleButton_add").show();
			$(".editEGW").animate({top:"-900px",opacity:"0"})
			$(".egwTit li").text("<%=rb.getString("EGWGuanLi")%>");
			$(".operationTit_Edit").text("<%=rb.getString("TianJia")%>");
		}
	}
	
	//右上角 添加 /关闭按钮 ，鼠标移入、移出事件 （下方显示提示文字 ）
	$(".addeGW").hover(function(){
		if($(".addeGW span").hasClass("addToeGW")){
			$(".operationTit").text("<%=rb.getString("GuanBi")%>");
			$(".operationTit").fadeIn(300);
		}else{
			$(".operationTit").text("<%=rb.getString("TianJia")%>");
			$(".operationTit").fadeIn(300);
		}
	},function(){
		$(".operationTit").fadeOut(300);
	})
	
	//右上角 添加 /关闭按钮 ，鼠标按下、放开事件 （按下瞬间不显示阴影 ）
	$(".addeGW").mousedown(function(){
		$(this).css("box-shadow","none")
	})
	$(".addeGW").mouseup(function(){
		$(this).css("box-shadow","3px 5px 17px rgba(51,153,204,0.3)")
	})
	
	//新建eGW  eGW信息和宏站配置切换 
	function TabsTurn(index,ele){
		if(index==2){
			var GateWayIp = $("#GateWayIP").val();
			if(getIseGWIp(GateWayIp)){
				$.messager.alert(TiShi, "<%=rb.getString("EGWBuNengWeiKong")%>");
				return false;
			}
			$(ele).addClass("active").siblings("li").removeClass("active"); 
			$(".NeweGWTab" + index).show().siblings("div").hide();
			$("#tablemacrocfglist_new").datagrid("reload");
		}else{
			$(ele).addClass("active").siblings("li").removeClass("active"); 
			$(".NeweGWTab" + index).show().siblings("div").hide();
		}
		$(window).resize();
		cancelAddMarco();
		closeNewMacroldSetting();
	}
	
	//修改eGW  eGW信息和宏站配置切换 
	function TabsTurnEdit(index,ele){
		$(ele).addClass("active").siblings("li").removeClass("active"); 
		$(".NeweGWTab" + index).show().siblings("div").hide();
		if(index==2){
			$("#tablemacrocfglist").datagrid("reload");
		}
		$(window).resize();
		cancelAddMarco();
		closeNewMacroldSetting();
	}
	
	//本地分流与话费服务器开关 
	function openSever(ele){
		if ($(ele).children().attr('isopen') == 'false') {
    		$(ele).children().attr('isopen','true').animate({left:'23px'},100);
    		$(ele).css('background-color','#66CC66');
    		$(".shuntAndCreditCont").slideToggle()
    	} else {
    		$(ele).children().attr('isopen','false').animate({left:'2px'},100);
            $(ele).css('background-color','#d7d7d7');
            $(".shuntAndCreditCont").slideToggle()
    	}
	}
	
	//功能开关 
	function openFunctions(ele){
		if ($(ele).children().attr('isopen') == 'false') {
    		$(ele).children().attr('isopen','true').animate({left:'24px'},100);
    		$(ele).css('background-color','#66CC66');
    	} else {
    		$(ele).children().attr('isopen','false').animate({left:'1px'},100);
            $(ele).css('background-color','#d7d7d7');
    	}
		
		//判断功能开关timer时间输入是否显示
		
		if(ele.id=="gwTimer-Edit-Swtich"&&$(ele).children().attr('isopen') == 'true'){
			$("#timers-edit-div").show();
			$("#timers-edit").val("").focus().select();
		}else if(ele.id=="gwTimer-Swtich"&&$(ele).children().attr('isopen') == 'true'){
			$("#timers-div").show();
			$("#timers").val("").focus().select();
		}else{
			$("#timers-edit-div,#timers-div").hide();
		}
	}
	
	//展开分流选项 （本地分流、全部分流 ）
	function openshuntChooseItem(e){
		var isShow = $(".shuntChooseItem").css("display");
		$(".shuntChooseItem").slideToggle(250);
		if(isShow == "none"){
			$(".chooseArrow").removeClass('egw-arrow-down').addClass('egw-arrow-up');
		}else{
			$(".chooseArrow").removeClass('egw-arrow-up').addClass('egw-arrow-down');
		}
		e.stopPropagation();
	}
	
	//选择分流方式 （1 -- 本地分流 ； 2 -- 全部分流  ）
	function chooseShunt(index){
		var radio = document.getElementsByName("shuntaddType");
		for(var i=0;i<radio.length;i++){
			if(radio[i].value==index){
				$("input[name='shuntaddType']").get(i).checked=true;
				break;
			}
		}
		//$("input[name='shuntaddType'][value="+index+"]").attr("checked",true);
		$(".shuntChooseItem").slideUp(250);
		$(".chooseArrow").removeClass('egw-arrow-up').addClass('egw-arrow-down');
		if(index == 1){
			$(".localShuntGroup").slideDown();
			$(".allShuntGroup").slideUp();
			//$("#shuntChooseText").text("本地分流");
		}else{
			$(".localShuntGroup").slideUp();
			$(".allShuntGroup").slideDown();
			//$("#shuntChooseText").text("全部分流");
		}
	}
	
	function chooseShuntEdit(index){
		var radio = document.getElementsByName("shuntType");
		for(var i=0;i<radio.length;i++){
			if(radio[i].value==index){
				$("input[name='shuntType']").get(i).checked=true;
				break;
			}
		}
		//$("input[name='shuntType'][value="+index+"]").attr("checked",true);
		$(".shuntChooseItem").slideUp(250);
		$(".chooseArrow").removeClass('egw-arrow-up').addClass('egw-arrow-down');
		if(index == 1){
			$(".localShuntGroup_Edit").slideDown();
			$(".allShuntGroup").slideUp();
			//$("#shuntChooseTextEdit").text("本地分流");
		}else{
			$(".localShuntGroup_Edit").slideUp();
			$(".allShuntGroup").slideDown();
			//$("#shuntChooseTextEdit").text("全部分流");
		}
	}
	
	function addLocalShuntGroup(){
		var GroupLength = $(".localShuntGroupItem-add").length;
		var newClone = $(".defaultlocalShuntGroup-add").clone();
    	newClone.find("input").each(function(){
    		$(this).val("");
    	})
		if(GroupLength >= 3){
	    	$(".defaultlocalShuntGroup-add .addLocalShuntGroup").hide();
	    	$(".defaultlocalShuntGroup-add .addLocalShuntGroup-dis").css("display","inline-block");
	    	newClone.children(".addLocalShuntGroup").replaceWith("<div class='subLocalShuntGroup defaultSub' onclick='subLocalShuntGroup(this)'></div>");
	    	newClone.remove(".addLocalShuntGroup");
		}else{
			$(".defaultlocalShuntGroup-add .addLocalShuntGroup").css("display","inline-block");
	    	$(".defaultlocalShuntGroup-add .addLocalShuntGroup-dis").hide();
	    	newClone.children(".addLocalShuntGroup").replaceWith("<div class='subLocalShuntGroup defaultSub' onclick='subLocalShuntGroup(this)'></div>");
		}
    	newClone.find(".localShuntNumber").text(GroupLength+1);
    	newClone.find(".checkTit").text("");
    	newClone.remove(".addLocalShuntGroup");
    	newClone.removeClass("defaultlocalShuntGroup-add");
    	$(".localShuntGroup").append(newClone);
	}
	
	function subLocalShuntGroup(ele){
		var GroupLength = $(".localShuntGroupItem-add").length;
		if(GroupLength < 5){
			$(".defaultlocalShuntGroup-add .addLocalShuntGroup").css("display","inline-block");
	    	$(".defaultlocalShuntGroup-add .addLocalShuntGroup-dis").hide();
		}
		$(ele).parent().remove();
	}
	
	function addLocalShuntGroup_Edit(){
		var GroupLength = $(".localShuntGroupItem-Edit").length;
		var newClone = $(".defaultlocalShuntGroup-Edit").clone();
    	newClone.find("input").each(function(){
    		$(this).val("");
    	})
		if(GroupLength >= 3){
	    	$(".defaultlocalShuntGroup-Edit .addLocalShuntGroup").hide();
	    	$(".defaultlocalShuntGroup-Edit .addLocalShuntGroup-dis").css("display","inline-block");
	    	newClone.children(".addLocalShuntGroup").replaceWith("<div class='subLocalShuntGroup defaultSub' onclick='subLocalShuntGroup_Edit(this)'></div>");
	    	newClone.remove(".addLocalShuntGroup");
		}else{
			$(".defaultlocalShuntGroup-Edit .addLocalShuntGroup").css("display","inline-block");
	    	$(".defaultlocalShuntGroup-Edit .addLocalShuntGroup-dis").hide();
	    	newClone.children(".addLocalShuntGroup").replaceWith("<div class='subLocalShuntGroup defaultSub' onclick='subLocalShuntGroup_Edit(this)'></div>");
		}
    	newClone.find(".localShuntNumber").text(GroupLength+1);
    	newClone.find(".checkTit").text("");
    	newClone.remove(".addLocalShuntGroup");
    	newClone.removeClass("defaultlocalShuntGroup-Edit");
    	$(".localShuntGroup_Edit").append(newClone);
	}

	function subLocalShuntGroup_Edit(ele){
		var GroupLength = $(".localShuntGroupItem-Edit").length;
		if(GroupLength < 5){
			$(".defaultlocalShuntGroup-Edit .addLocalShuntGroup").css("display","inline-block");
	    	$(".defaultlocalShuntGroup-Edit .addLocalShuntGroup-dis").hide();
		}
		$(ele).parent().remove();
	}
	
	
	function newMacroldSetting(key){
		if(key == 'edit'){
			var GateWayIP_Edit = $("#GateWayIP_Edit").val();
			if(getIseGWIp(GateWayIP_Edit)){
				$.messager.alert(TiShi, "<%=rb.getString("EGWBuNengWeiKong")%>");
				return false;
			}
			$("#Edit-GatwayName").val($("#GateWayName_Edit").val());//zss
			$("#Edit-GatwayIP").val($("#GateWayIP_Edit").val());
			$("#Edit-GatwayPort").val($("#GateWayPort_Edit").val());
			$("#Edit-MacroId,#Edit-TAC").val("");
			$("#operSaveType").val("EDIT");
			setTimeout(function(){
				$("#tablemmelist").datagrid("loadData",{rows:[]});
			},100);
			//$("#tablemmelist").datagrid("loadData",{rows:[]});
		}else if(key == "editAdd"){
			$("#macrotitle").text("<%=rb.getString("XinJianHongZhan")%>");
			$("#Edit-MacroIdCheckSpan,#Edit-TACCheckSpan").text("");
			$("#Edit-MacroId,#Edit-TAC").val("");
			var GateWayIP_Edit = $("#GateWayIP_Edit").val();
			if(getIseGWIp(GateWayIP_Edit)){
				$.messager.alert(TiShi, "<%=rb.getString("EGWBuNengWeiKong")%>");
				return false;
			}
			$("#Edit-GatwayName").val($("#GateWayName_Edit").val());//zss
			$("#Edit-GatwayIP").val($("#GateWayIP_Edit").val());
			$("#Edit-GatwayPort").val($("#GateWayPort_Edit").val());
			$("#Edit-MacroId,#Edit-TAC").val("");
			$("#operSaveType").val("ADD");
			setTimeout(function(){
				$("#tablemmelist").datagrid("loadData",{rows:[]});
			},100);
		}else if(key == 'newAdd'){
			var GateWayIp = $("#GateWayIP").val();
			if(getIseGWIp(GateWayIp)){
				$.messager.alert(TiShi, "<%=rb.getString("EGWBuNengWeiKong")%>");
				return false;
			}
			$("#New-GatwayName").val($("#GateWayName").val());//zss
			$("#New-GatwayIP").val($("#GateWayIP").val());
			$("#New-GatwayPort").val($("#GateWayPort").val());
			$("#operSaveType-add").val("ADD");
			setTimeout(function(){
				$("#addMacroldTable").datagrid("loadData",{rows:[]});
			},100);
		}else if(key == 'newEdit'){
			var GateWayIp = $("#GateWayIP").val();
			if(getIseGWIp(GateWayIp)){
				$.messager.alert(TiShi, "<%=rb.getString("EGWBuNengWeiKong")%>");
				return false;
			}
			$("#New-GatwayName").val($("#GateWayName").val());//zss
			$("#New-GatwayIP").val($("#GateWayIP").val());
			$("#New-GatwayPort").val($("#GateWayPort").val());
			$("#operSaveType-add").val("EDIT");
			setTimeout(function(){
				$("#addMacroldTable").datagrid("loadData",{rows:[]});
			},100);
		}
		$("#New-MacroId,#New-MacroId-Hidden").val("");
		$("#new-TAC,#new-TAC-Hidden").val("");
		$(".newMacroldSetting").animate({right:'0px'},500);
		//wjw优化
		//获取当前窗口宽度根据窗口大小判断
		/* var loadwidth = $(document).width();
		if(loadwidth>=1401){
			$(".newMacroldSetting h3").css("width","1000px")
			$(".newMacroldSetting").animate({right:'0px'},500);
		}else{
			$(".newMacroldSetting h3").css("width","800px")
			$(".newMacroldSetting").animate({right:'-200px'},500);
			//$(".eGWBasicInfoItemDiv").addclass("eGWBasicInfoItemDivNoMarginRight");
		} */
		//$(".newMacroldSetting").animate({right:'0px'},500);
		
		//动态监听文档窗口宽度
		/* 	$(window).resize(function(){
		var nowwidth = $(document).width();
		if(nowwidth>=1401){
			
			$(".newMacroldSetting h3").css("width","1000px")
			$(".newMacroldSetting").animate({right:'0px'},500);
			
		}else{
			
			$(".newMacroldSetting h3").css("width","800px")
			$(".newMacroldSetting").animate({right:'-200px'},500);
			$(".eGWBasicInfoItemDiv").addclass("eGWBasicInfoItemDivNoMarginRight");
		
		}
	}) */
	
	}
	
	//结束
	function closeNewMacroldSetting(){
		$(".newMacroldSetting").animate({right:'-1200px'},500);
		cancelAddMarco();
	}
	
	function getIseGWIp(gwip){
		var flag=true;
		$.ajax({
			type: "post",
			url: "${ctx}/eGW/egwManage/getIseGWIp.action?TimeZone="+timeZone,
			data: {gwip:gwip},
			async: false,
			dataType:"json",
			success: function(data) {
				if(data["message"]=="1"){
					flag=false;
				}
			}
		});
		return flag;
	}
	
	//点击添加链路信息触发
	$(".addLink").click(function(){
		if(this.id=="addLink_New"){//添加配置时添加链路操作
			$('#New-MacroIdCheckSpan,#new-TACCheckSpan').text("");
			if($("#New-MacroId").val()==""){
				$('#New-MacroIdCheckSpan').text("<%=rb.getString("BuNengWeiKong")%>");
				$("#New-MacroId").focus().select();
				return false;
			}
			if($("#new-TAC").val()==""){
				$('#new-TACCheckSpan').text("<%=rb.getString("BuNengWeiKong")%>");
				$("#new-TAC").focus().select();
				return false;
			}
			var GroupLength = $(".LinkInfoDefault").length;
			$(".LinkInfoDefault").find("img").each(function(index){
				if(index==1){
					subMarroIP(this);
				}
			});
			var GroupLength = $(".LinkInfommeIp").length;
			$(".LinkInfommeIp").find("img").each(function(index){
				if(index==1){
					subMarroIP(this);
				}
			});
			//默认添加链路开关为开启状态
			$("#upLinkEnable-Add-Switch").children().attr('isopen','true').animate({left:'24px'},100);
    		$("#upLinkEnable-Add-Switch").css('background-color','#66CC66');
		}else if(this.id=="addLink_Edit"){//修改配置时添加链路操作
			var MacroId = $("#Edit-MacroId").val();
			$('#Edit-MacroIdCheckSpan,#Edit-TACCheckSpan').text("");
			if(MacroId==""){
				$('#Edit-MacroIdCheckSpan').text("<%=rb.getString("BuNengWeiKong")%>");
				$("#Edit-MacroId").focus().select();
				return false;
			}
			var TAC = $("#Edit-TAC").val();
			if(TAC==""){
				$('#Edit-TACCheckSpan').text("<%=rb.getString("BuNengWeiKong")%>");
				$("#Edit-TAC").focus().select();
				return false;
			}
			var GroupLength = $(".LinkInfoDefault-Edit").length;
			$(".LinkInfoDefault-Edit").find("img").each(function(index){
				if(index==1){
					subMarroIP_Edit(this);
				}
			});
			var GroupLength = $(".LinkInfommeIp-Edit").length;
			$(".LinkInfommeIp-Edit").find("img").each(function(index){
				if(index==1){
					subMarroIP_Edit(this);
				}
			});
			//默认添加链路开关为开启状态
			$("#upLinkEnable-Edit-Switch").children().attr('isopen','true').animate({left:'24px'},100);
    		$("#upLinkEnable-Edit-Switch").css('background-color','#66CC66');
		}
		$(".addLinkInfoItem").slideToggle(250);
		$("#opertypemme,#opertypemme-add").val("add");
		$("#macroIp-edit,#macroPort-edit,#mmeIp-edit,#mmePort-edit").val("");
		$("#macroIp-add,#macroPort-add,#mmeIp-add,#mmePort-add").val("");
	});
	
	function addMIP(ele){
		var newClone = $(ele).parent().clone();
    	newClone.find("input").each(function(){
    		$(this).val("");
    	})	
	    newClone.children(".addMacroIP").replaceWith("<img onclick='subMarroIP(this)' class='subMacroIP' src='${ctx}/css/images/bi/IP-sub.png'/>");
    	$(ele).parent().parent().append(newClone)
    	$(ele).replaceWith("<img  src='${ctx}/css/images/bi/IP-add-dis.png'/>");
	}
	
	function subMarroIP(ele){
		$(ele).parent().parent().children("div").first().find("img").replaceWith("<img class='addMacroIP' onclick='addMIP(this)' src='${ctx}/css/images/bi/IP-add.png'/>");
		$(ele).parent().remove(); 
	}
	
	//修改面板中对MME进行修改面板操作
	function addMIP_Edit(ele){
		var GroupLength = $(".LinkInfoDefault-Edit").length;
		if(GroupLength<=2){
			var newClone = $(ele).parent().clone();
	    	newClone.find("input").each(function(){
	    		$(this).val("");
	    	})
		    newClone.children(".addMacroIP-Edit").replaceWith("<img onclick='subMarroIP_Edit(this)' class='subMacroIP' src='${ctx}/css/images/bi/IP-sub.png'/>");
	    	$(ele).parent().parent().append(newClone)
	    	$(ele).replaceWith("<img src='${ctx}/css/images/bi/IP-add-dis.png'/>");
		}
	}
	//修改面板中对MME进行修改面板操作
	function subMarroIP_Edit(ele){
		$(ele).parent().parent().children("div").first().find("img").replaceWith("<img class='addMacroIP-Edit' onclick='addMIP_Edit(this)' src='${ctx}/css/images/bi/IP-add.png'/>");
		$(ele).parent().remove(); 
	}
	
	//关闭链路操作面板
	function cancelAddMarco(){
		$(".addLinkInfoItem").slideUp(300);
		var inputInfo = $(".addLinkInfoItem").clone();
		inputInfo.find("input").each(function(){
			$("#"+this.id+"CheckSpan").text("");
			$("#"+this.id).val("");
		});
	}
	
	function getParamsBeforeLoad(param){}
	function getParamsMacrocfgLoad(param){}
	function getParamsMMELoad(param){}
	
	//添加egw信息操作
	function addGateWaySetUpInfo(){
		var inputInfo = $(".eGWBasicInfo").clone();
		var flag=false;
		$("#timersCheckSpan").text("");
		inputInfo.find("input").each(function(){
			if(this.value==""){
				$("#"+this.id+"CheckSpan").text("<%=rb.getString("BuNengWeiKong")%>");
				$("#"+this.id).focus().select();
				flag=true;
				return false;
			}else{
				if((!checkIpfun(this.id,this.value)&&this.getAttribute("ftype")=="IP")
						||(!checkPort(this.id,this.value)&&this.getAttribute("ftype")=="Port")){
					flag=true;
					return false;
				}else{
					$("#"+this.id+"CheckSpan").text("");
				}
			}
		});
		if(checkPlmn("PLMN",$("#PLMN").val())){
			return;
		}
		var gw = {};
		gw["gwName"]=$("#GateWayName").val();
		gw["gwIp"]=$("#GateWayIP").val();
		gw["gwPort"]=$("#GateWayPort").val();
		var comm = {};
		comm["plmn"]=$("#PLMN").val();
		
		var addrinfo={};
		var accessArr=[];
		var enbAccessIp={};
		var enbAccessIps={};
		enbAccessIp["tag"]="A";
		enbAccessIp["accessIp"]=$("#eNBIp").val();
		enbAccessIp["accessPort"]=$("#eNBPort").val();
		accessArr[0]=enbAccessIp;
		enbAccessIps["enbAccessIp"]=accessArr;
		if(!$.isEmptyObject(enbAccessIps)){
			addrinfo["enbAccessIps"]=enbAccessIps;
		}
		
		var s1UEnbIpArr=[];
		var S1UEnbIp={};
		var S1UEnbIps={};
		S1UEnbIp["tag"]="A";
		S1UEnbIp["S1UEnbIp"]=$("#S1UeNBIP").val();
		s1UEnbIpArr[0]=S1UEnbIp;
		S1UEnbIps["enbIpInfo"]=s1UEnbIpArr;
		if(!$.isEmptyObject(S1UEnbIps)){
			addrinfo["S1UEnbIps"]=S1UEnbIps;
		}
		
		var s1UMmeIpArr=[];
		var S1UMmeIp={};
		var S1UMmeIps={};
		S1UMmeIp["tag"]="A";
		S1UMmeIp["S1UMmeIp"]=$("#S1UMMEIP").val();
		s1UMmeIpArr[0]=S1UMmeIp;
		S1UMmeIps["mmeIpInfo"]=s1UMmeIpArr;
		if(!$.isEmptyObject(S1UMmeIps)){
			addrinfo["S1UMmeIps"]=S1UMmeIps;
		}
		
		//话单配置信息
		//if ($("#lfpSwitch").attr('isopen') == 'true'){
			var cdr={};
			if($("#CDR-Generate-IP").val()!=""){
				if(!checkIpfun('CDR-Generate-IP',$("#CDR-Generate-IP").val())){
					return false;
				}
				cdr["cdrGenerateIp"]=$("#CDR-Generate-IP").val();
			}
			if($("#CDR-Service-IP").val()!=""){
				if(!checkIpfun('CDR-Service-IP',$("#CDR-Service-IP").val())){
					return false;
				}
				cdr["cdrServIp"]=$("#CDR-Service-IP").val();
			}
			if($("#CDR-Number").val()!=""){
				cdr["cdrNum"]=$("#CDR-Number").val();
			}
			if(!$.isEmptyObject(cdr)){
				addrinfo["chargeService"]=cdr;
			}
			
			var lfps={};
			var forwardPolicy = "";
			//$("input[name='shuntType'][checked]").val();
			var radio = document.getElementsByName("shuntType");
			for(var i=0;i<radio.length;i++){
				if(radio[i].checked){
					forwardPolicy=radio[i].value;
					break;
				}
			}
			/*
			lfps["forwardPolicy"]=forwardPolicy;//0-全部 1-本地 
			if(forwardPolicy=='1'){
				var lfp=[];
				$(".localShuntGroupItem-add").each(function(index,element){
					var lfpConfig={};
					var gwDNIP = $(this).find("input[name='gwDNIP']").val();
					if(gwDNIP==""){
						$(this).find("span[name='gwDNIPCheckSpan']").text("It cannot be null.");
						$(this).find("input[name='gwDNIP']").focus().select();
						flag=true;
						return false;
					}else{
						if(!isValidIP(gwDNIP)){
							$(this).find("span[name='gwDNIPCheckSpan']").text("Ip format error.");
							$(this).find("input[name='gwDNIP']").focus().select();
							flag=true;
							return false;
						}else{
							$(this).find("span[name='gwDNIPCheckSpan']").text("");
						}
					}
					var gwDNM = $(this).find("input[name='gwDNM']").val();
					if(gwDNM==""){
						$(this).find("span[name='gwDNMCheckSpan']").text("It cannot be null.");
						$(this).find("input[name='gwDNM']").focus().select();
						flag=true;
						return false;
					}else{
						if(!isValidIP(gwDNM)){
							$(this).find("span[name='gwDNMCheckSpan']").text("mark format error.");
							$(this).find("input[name='gwDNM']").focus().select();
							flag=true;
							return false;
						}else{
							$(this).find("span[name='gwDNMCheckSpan']").text("");
						}
					}
					lfpConfig["tag"]="A";
					lfpConfig["netIp"]=gwDNIP;
					lfpConfig["netMask"]=gwDNM;
					lfp[index]=lfpConfig;
				});
				lfps["localForwardPolicy"]=lfp;
			}
			comm["localForwardPolicys"]=lfps;
		}
		*/
		comm["AddrInfo"]=addrinfo;	
		var fswitch={};
		/*第二版在加
		fswitch["enbDynamicReg"]=$("#gwDynamicreg").attr('isopen') == 'true'?"enable":"disable";
		fswitch["dataRelay"]=$("#gwDatarelay").attr('isopen') == 'true'?"enable":"disable";
		fswitch["sigFw"]=$("#gwSigfw").attr('isopen') == 'true'?"enable":"disable";
		fswitch["upLinkSelect"]=$("#gwUplinkselect").attr('isopen') == 'true'?"enable":"disable";
		fswitch["eGWlog"]=$("#gweGWLog").attr('isopen') == 'true'?"enable":"disable";
		fswitch["opType"]=$("#gwTimer").attr('isopen') == 'true'?"enable":"disable";
		if($("#gwTimer").attr('isopen')=="true"){
			var timers = $("#timers").val();
			if(timers==""){
				$("#timersCheckSpan").text("timers cannot be null.");
				$("#timers").focus().select();
				flag=true;
			}
			fswitch["timer"]=timers;
		}*/
		if(!$.isEmptyObject(fswitch)){
			comm["funcSwitch"]=fswitch;
		}
		
		gw["CommonMgmt"]=comm;
		if(flag){
			return false;
		}	
		var json=JSON.stringify(gw);
		//$.messager.alert(TiShi, json);
		if(!getIseGWIp($("#GateWayIP").val())){
			$("#GateWayIPCheckSpan").text("<%=rb.getString("WangGuanYiCunZaiQingChongXinShuRu")%>");
			$("#GateWayIP").focus().select();
			return false;
		}
		$("#winLoadingPro").window("open");
 		$.ajax({
			type: "post",
			url: "${ctx}/eGW/egwManage/setEgwJsonToXml.action?TimeZone="+timeZone, 
			data: {jsonStr:json},
			async: true,
			dataType:"json",
			success: function(data) {
				$("#winLoadingPro").window("close");
				if (data["causeCode"]=="1"||data["causeCode"]==1){
					$("#tableGateWayList").datagrid("reload");
					TabsTurn(2,$("#macroid_li"));//gateway添加成功后自动跳转到宏站添加面板
					getGateWayAddInfo(gw["gwIp"],gw["gwPort"],gw["gwName"]);
					closeNewMacroldSetting();
					rotateR();
					//$.messager.alert(TiShi, "操作成功.");
				}else if(data["causeCode"]=="0"||data["causeCode"]==0||data["causeCode"]==undefined){
					$.messager.alert(TiShi, "<%=rb.getString("CaoZuoShiBai")%>");
				}else if(data["causeCode"]=="2"||data["causeCode"]==2){
					if(data["name"]=="accessIp"){
						$("#eNBIp").focus().select();
						$("#eNBIpCheckSpan").text("<%=rb.getString("IPBuKeYong")%>");
					}else if(data["name"]=="S1UEnbIp"){
						$("#S1UeNBIP").focus().select();
						$("#S1UeNBIPCheckSpan").text("<%=rb.getString("IPBuKeYong")%>");
					}else if(data["name"]=="S1UMmeIp"){
						$("#S1UMMEIP").focus().select();
						$("#S1UMMEIPCheckSpan").text("<%=rb.getString("IPBuKeYong")%>");
					}else{
						$.messager.alert(TiShi, "<%=rb.getString("CaoZuoShiBai")%>");
					}
				}else{
					$.messager.alert(TiShi, data["reasonCode"]);
				}
			}
		});
	}
	
	function editGateWaySetUpInfo(){
		var inputInfo = $(".eGWBasicInfo-Edit").clone();
		var flag=false;
		$("#timers-editCheckSpan").text("");
		inputInfo.find("input").each(function(){
			if(this.value==""){
				$("#"+this.id+"CheckSpan").text("<%=rb.getString("BuNengWeiKong")%>");
				$("#"+this.id).focus().select();
				flag=true;
				return false;
			}else{
				if((!checkIpfun(this.id,this.value)&&this.getAttribute("ftype")=="IP")
						||(!checkPort(this.id,this.value)&&this.getAttribute("ftype")=="Port")){
					flag=true;
					return false;
				}else{
					$("#"+this.id+"CheckSpan").text("");
				}
			}
		});
		if(checkPlmn("PLMN_Edit",$("#PLMN_Edit").val())){
			return;
		}
		var gw = {};
		gw["gwName"]=$("#GateWayName_Edit").val();
		gw["gwIp"]=$("#GateWayIP_Edit").val();
		gw["gwPort"]=$("#GateWayPort_Edit").val();
		var comm = {};
		if(dataMap["plmn"]!=$("#PLMN_Edit").val()){
			comm["plmn"]=$("#PLMN_Edit").val();
		}
		var addrinfo={};
		
		var enbAccessIpsinfo = dataMap["enbAccessIps"]==undefined?"":dataMap["enbAccessIps"][0]["accessIp"];
		var accessPortinfo = dataMap["enbAccessIps"]==undefined?"":dataMap["enbAccessIps"][0]["accessPort"];
		
		//异常情况处理基站接入信息，查询为空时进行添加操作
		if(enbAccessIpsinfo==""){
			var accessArr=[];
			var enbAccessIp={};
			var enbAccessIps={};
				enbAccessIp["tag"]="A";
				enbAccessIp["accessIp"]=$("#eNBIp_Edit").val();
				enbAccessIp["accessPort"]=$("#eNBPort_Edit").val();
				accessArr[0]=enbAccessIp;
				enbAccessIps["enbAccessIp"]=accessArr;
				if(!$.isEmptyObject(enbAccessIps)){
					addrinfo["enbAccessIps"]=enbAccessIps;
				}
		}else{
			if((enbAccessIpsinfo!=$("#eNBIp_Edit").val()&&$("#eNBIp_Edit").val()!="")
					||(accessPortinfo!=$("#eNBPort_Edit").val()&&$("#eNBPort_Edit").val()!="")){
				var accessArr=[];
				var enbAccessIp={};
				var enbAccessIps={};
					enbAccessIp["tag"]="M";
					enbAccessIp["currentAccessIp"]=enbAccessIpsinfo;
					enbAccessIp["currentAccessPort"]=accessPortinfo;
					enbAccessIp["accessIp"]=$("#eNBIp_Edit").val();
					enbAccessIp["accessPort"]=$("#eNBPort_Edit").val();
					accessArr[0]=enbAccessIp;
					enbAccessIps["enbAccessIp"]=accessArr;
					if(!$.isEmptyObject(enbAccessIps)){
						addrinfo["enbAccessIps"]=enbAccessIps;
					}
			}
		}
		
		var S1UEnbIpsinfo = dataMap["S1UEnbIps"]==undefined?"":dataMap["S1UEnbIps"][0]["S1UEnbIp1"];
		if(S1UEnbIpsinfo==""){
			//异常修改查询为空时，进行添加操作
			var s1UEnbIpArr=[];
			var S1UEnbIp={};
			var S1UEnbIps={};
			S1UEnbIp["tag"]="A";
			S1UEnbIp["S1UEnbIp"]=$("#S1UeNBIP_Edit").val();
			s1UEnbIpArr[0]=S1UEnbIp;
			S1UEnbIps["enbIpInfo"]=s1UEnbIpArr;
			if(!$.isEmptyObject(S1UEnbIps)){
				addrinfo["S1UEnbIps"]=S1UEnbIps;
			}
		}else{
			if((S1UEnbIpsinfo!=$("#S1UeNBIP_Edit").val()&&$("#S1UeNBIP_Edit").val()!="")){
				var s1UEnbIpArr=[];
				var S1UEnbIp={};
				var S1UEnbIps={};
				S1UEnbIp["tag"]="M";
				S1UEnbIp["currentS1UEnbIp"]=S1UEnbIpsinfo;
				S1UEnbIp["S1UEnbIp"]=$("#S1UeNBIP_Edit").val();
				s1UEnbIpArr[0]=S1UEnbIp;
				S1UEnbIps["enbIpInfo"]=s1UEnbIpArr;
				if(!$.isEmptyObject(S1UEnbIps)){
					addrinfo["S1UEnbIps"]=S1UEnbIps;
				}
			}
		}
		
		var S1UMmeIpsinfo = dataMap["S1UMmeIps"]==undefined?"":dataMap["S1UMmeIps"][0]["S1UMmeIp1"];
		if(S1UMmeIpsinfo==""){
			//异常修改查询为空时，进行添加操作
			var s1UMmeIpArr=[];
			var S1UMmeIp={};
			var S1UMmeIps={};
			S1UMmeIp["tag"]="A";
			S1UMmeIp["S1UMmeIp"]=$("#S1UMMEIP_Edit").val();
			s1UMmeIpArr[0]=S1UMmeIp;
			S1UMmeIps["mmeIpInfo"]=s1UMmeIpArr;
			if(!$.isEmptyObject(S1UMmeIps)){
				addrinfo["S1UMmeIps"]=S1UMmeIps;
			}
		}else{
			if((S1UMmeIpsinfo!=$("#S1UMMEIP_Edit").val()&&$("#S1UMMEIP_Edit").val()!="")){
				var s1UMmeIpArr=[];
				var S1UMmeIp={};
				var S1UMmeIps={};
				S1UMmeIp["tag"]="M";
				S1UMmeIp["currentS1UMmeIp"]=S1UMmeIpsinfo;
				S1UMmeIp["S1UMmeIp"]=$("#S1UMMEIP_Edit").val();
				s1UMmeIpArr[0]=S1UMmeIp;
				S1UMmeIps["mmeIpInfo"]=s1UMmeIpArr;
				if(!$.isEmptyObject(S1UMmeIps)){
					addrinfo["S1UMmeIps"]=S1UMmeIps;
				}
			}
		}
		
		//话单配置信息
		//if ($("#lfpSwitch_Edit").attr('isopen') == 'true'){
		var cdr={};
		if($("#CDR-Generate-IP-Edit").val()!=""||$("#CDR-Service-IP-Edit").val()!=""||$("#CDR-Number-Edit").val()!=""){
			if(!checkIpfun('CDR-Generate-IP-Edit',$("#CDR-Generate-IP-Edit").val())){
				return false;
			}
			cdr["cdrGenerateIp"]=$("#CDR-Generate-IP-Edit").val();
			if(!checkIpfun('CDR-Service-IP-Edit',$("#CDR-Service-IP-Edit").val())){
				return false;
			}
			cdr["cdrServIp"]=$("#CDR-Service-IP-Edit").val();
			cdr["cdrNum"]=$("#CDR-Number-Edit").val();
		}
		if(!$.isEmptyObject(cdr)){
			addrinfo["chargeService"]=cdr;
		}
		var lfps={};
			/*
			//获取分流策略
			var forwardPolicy = "";
			//$("input[name='shuntType'][checked]").val();
			var radio = document.getElementsByName("shuntType");
			for(var i=0;i<radio.length;i++){
				if(radio[i].checked){
					forwardPolicy=radio[i].value;
					break;
				}
			}
			lfps["forwardPolicy"]=forwardPolicy;//0-全部 1-本地  2-无分流
			if(forwardPolicy=='1'){
				var lfp=[];
				var num=0;
				$(".localShuntGroupItem-Edit").each(function(index,element){
					var lfpConfig={};
					var gwDNIP = $(this).find("input[name='gwDNIP_Edit']").val();
					if(gwDNIP==""){
						$(this).find("span[name='gwDNIP_EditCheckSpan']").text("It cannot be null.");
						$(this).find("input[name='gwDNIP_Edit']").focus().select();
						flag=true;
						return false;
					}else{
						if(!isValidIP(gwDNIP)){
							$(this).find("span[name='gwDNIP_EditCheckSpan']").text("Ip format error.");
							$(this).find("input[name='gwDNIP_Edit']").focus().select();
							flag=true;
							return false;
						}else{
							$(this).find("span[name='gwDNIP_EditCheckSpan']").text("");
						}
					}
					var gwDNM = $(this).find("input[name='gwDNM_Edit']").val();
					if(gwDNM==""){
						$(this).find("span[name='gwDNM_EditCheckSpan']").text("It cannot be null.");
						$(this).find("input[name='gwDNM_Edit']").focus().select();
						flag=true;
						return false;
					}else{
						if(!isValidIP(gwDNM)){
							$(this).find("span[name='gwDNM_EditCheckSpan']").text("mark format error.");
							$(this).find("input[name='gwDNM_Edit']").focus().select();
							flag=true;
							return false;
						}else{
							$(this).find("span[name='gwDNM_EditCheckSpan']").text("");
						}
					}
					//判断原有话单服务是否存在
					var cdrnum=0;
					var oldnetIp="",oldnetMask="";
					var lennum = $(".localShuntGroupItem-Edit").length;
					if(dataMap["localForwardPolicys"]!=undefined&&dataMap["localForwardPolicys"]!=""){
						cdrnum=dataMap["localForwardPolicys"].length;
						if(index<cdrnum){
							oldnetIp = dataMap["localForwardPolicys"][index]["netIp"];
							oldnetMask = dataMap["localForwardPolicys"][index]["netMask"];
						}
					}
					var gwDNIP_Edit = $(this).find("input[name='gwDNIP_Edit']").val();
					var gwDNM_Edit = $(this).find("input[name='gwDNM_Edit']").val();
					//判断是否有修改操作
					if(index<cdrnum){
						lfpConfig["tag"]="M";
						lfpConfig["currentNetIp"]=oldnetIp
						lfpConfig["currentNetMask"]=oldnetMask
						lfpConfig["netIp"]=gwDNIP_Edit
						lfpConfig["netMask"]=gwDNM_Edit
						lfp[num]=lfpConfig;
						num++;
					}
					//判断是否有添加操作
					if(lennum>cdrnum&&(index+1)>cdrnum){
						lfpConfig["tag"]="A";
						lfpConfig["netIp"]=gwDNIP_Edit
						lfpConfig["netMask"]=gwDNM_Edit
						lfp[num]=lfpConfig;
						num++;
					}
					//判断是否有删除操作
					if(lennum<cdrnum&&index<cdrnum){
						for(var i=lennum+1;i<=cdrnum;i++){
							lfpConfig["tag"]="D";
							lfpConfig["currentNetIp"]=dataMap["localForwardPolicys"][i]["netIp"]
							lfpConfig["currentNetMask"]=dataMap["localForwardPolicys"][i]["netMask"]
							lfp[num]=lfpConfig;
							num++;
						}
					}
				});
				if(lfp.length!=0){
					lfps["localForwardPolicy"]=lfp;
				}
			}
			if(!$.isEmptyObject(lfps)){
				comm["localForwardPolicys"]=lfps;
			}*/
		//}
		if(!$.isEmptyObject(addrinfo)){
			comm["AddrInfo"]=addrinfo;
		}
		var fswitch={};
		var enbDynamicReg=$("#gwDynamicreg-Edit").attr('isopen') == 'true'?"enable":"disable";
		var dataRelay=$("#gwDatarelay-Edit").attr('isopen') == 'true'?"enable":"disable";
		var sigFw=$("#gwSigfw-Edit").attr('isopen') == 'true'?"enable":"disable";
		var upLinkSelect=$("#gwUplinkselect-Edit").attr('isopen') == 'true'?"enable":"disable";
		var eGWlog=$("#gweGWLog-Edit").attr('isopen') == 'true'?"enable":"disable";
		var opType=$("#gwTimer-Edit").attr('isopen') == 'true'?"enable":"disable";
		var timer = $("#timers-edit").val();
		if((enbDynamicReg==dataMap["enbDynamicReg"]&&dataRelay==dataMap["dataRelay"]&&dataMap["sigFw"]==sigFw
				&&upLinkSelect==dataMap["upLinkSelect"]&&eGWlog==dataMap["eGWlog"]&&opType==dataMap["opType"])){
		}else{
			/*第一版本不需要，等待第二版才加
			fswitch["enbDynamicReg"]=enbDynamicReg;
			fswitch["dataRelay"]=dataRelay;
			fswitch["sigFw"]=sigFw;
			fswitch["upLinkSelect"]=upLinkSelect;
			fswitch["eGWlog"]=eGWlog;
			fswitch["opType"]=opType;
			
			if(opType == 'enable'){
				if(timer==""){
					$("#timers-edit").focus().select();
					$("#timers-editCheckSpan").text("timers cannot be null.");
					flag=true;
				}
				fswitch["timer"]=timer;
			}
			*/
			if(!$.isEmptyObject(fswitch)){
				comm["funcSwitch"]=fswitch;
			}
		}
		
		gw["CommonMgmt"]=comm;
		if(flag){
			return false;
		}
		var json=JSON.stringify(gw);
		$("#winLoadingPro").window("open");
 		$.ajax({
			type: "post",
			url: "${ctx}/eGW/egwManage/setEgwJsonToXml.action?TimeZone="+timeZone,
			data: {jsonStr:json},
			async: true,
			dataType:"json",
			success: function(data) {
				$("#winLoadingPro").window("close");
				if (data["causeCode"]=="1"||data["causeCode"]==1){
					getGateWayEditInfo(gw["gwIp"],gw["gwPort"],gw["gwName"]);
					closeNewMacroldSetting();
					$("#tableGateWayList").datagrid("reload");
					$.messager.alert(TiShi, "<%=rb.getString("CaoZuoChengGong")%>");
				}else if(data["causeCode"]=="0"||data["causeCode"]==0||data["causeCode"]==undefined){
					$("#tableGateWayList").datagrid("reload");
					$.messager.alert(TiShi, "<%=rb.getString("CaoZuoShiBai")%>");
				}else if(data["causeCode"]=="2"||data["causeCode"]==2){
					if(data["name"]=="accessIp"){
						$("#eNBIp_Edit").focus().select();
						$("#eNBIp_EditCheckSpan").text("<%=rb.getString("IPBuKeYong")%>");
					}else if(data["name"]=="S1UEnbIp"){
						$("#S1UeNBIP_Edit").focus().select();
						$("#S1UeNBIP_EditCheckSpan").text("<%=rb.getString("IPBuKeYong")%>");
					}else if(data["name"]=="S1UMmeIp"){
						$("#S1UMMEIP_Edit").focus().select();
						$("#S1UMMEIP_EditCheckSpan").text("<%=rb.getString("IPBuKeYong")%>");
					}else{
						$.messager.alert(TiShi, "<%=rb.getString("CaoZuoShiBai")%>");
					}
				}else{
					$.messager.alert(TiShi, data["reasonCode"]);
				}
			}
		});
	}
	
	function operFormatter(value, rowData, rowIndex){
		var res="";
			res="<div class='operationDiv operation_edit' title='<%=rb.getString("XiuGai")%>' onclick='editGateWayInfo(true,\"" + rowData.GW_IP + "\",\""+rowData.GW_PORT+"\",\""+rowData.GW_NAME+"\")'></div>";
			res+="<div class='operationDiv operation_delete' style='margin-left: 10px' title='<%=rb.getString("ShanChu")%>' onclick='delGateWayInfo(true,\"" + rowData.GW_IP + "\",\""+rowData.GW_PORT+"\",\""+rowData.GW_NAME+"\")'></div>";
			res+="<div class='operationDiv operation_reboot' style='margin-left: 10px' title='<%=rb.getString("ChongQi")%>' onclick='rebootGateWayInfo(this,\"" + rowData.GW_IP + "\",\""+rowData.GW_PORT+"\",\""+rowData.GW_NAME+"\")'></div>";
			return res;
	}
	
	//重启gateway
	function rebootGateWayInfo(obj,gwip,gwport,gwname){
		$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingChongQiWangGuan")%>", function(r) {
	        if (r) {
	        	//$(obj).addClass("icon-activation-disabled");
	    		$.ajax({
	    			type: "post",
	    			url: "${ctx}/eGW/egwManage/rebootGateWay.action?TimeZone="+timeZone,
	    			data: {gwip:gwip,gwport:gwport,gwname:gwname},
	    			async: true,
	    			dataType:"json",
	    			success: function(data) {
	    				$(obj).removeClass("icon-activation-disabled");
	    				$(obj).addClass("icon-activation");
	    				
	    			},
	    			error:function(xmlhttprequest,textstatus,errorThrown){
	    				$(obj).removeClass("icon-activation-disabled");
	    				$(obj).addClass("icon-activation");
	    			}
	    		});
	        }
	    }).addClass('seriousConfirm'); 
		
	}
	
	function editGateWayInfo(flag,gwip,gwport,gwname){
		$(".shuntAndCreditCont").show();
		getGateWayEditInfo(gwip,gwport,gwname);
	}
	
	//获取egw信息，接口调取
	var dataMap={};
	function getGateWayEditInfo(gwip,gwport,gwname){
		$("#winLoadingPro").window("open");
		$.ajax({
			type: "post",
			url: "${ctx}/eGW/egwManage/queryAllCfgRsp.action?TimeZone="+timeZone,
			data: {gwip:gwip,gwport:gwport,gwname:gwname},
			async: true,
			dataType:"json",
			success: function(data) {
				$("#winLoadingPro").window("close");
				if(data["flag"]==0){
					$.messager.alert(TiShi, "<%=rb.getString("QingQiuYiChang")%>");
					return false;
				}
				rotateR_Edit();
				dataMap=data;
				$("#GateWayName_Edit").val(gwname);
				$("#GateWayIP_Edit").val(gwip);
				$("#GateWayPort_Edit").val(gwport);
				$("#PLMN_Edit").val(data["plmn"]);

				if(data["enbAccessIps"]!=undefined&&data["enbAccessIps"]!=""){
					$("#eNBIp_Edit").val(data["enbAccessIps"][0]["accessIp"]);
					$("#eNBPort_Edit").val(data["enbAccessIps"][0]["accessPort"]);
				}
				if(data["S1UEnbIps"]!=undefined&&data["S1UEnbIps"]!=""){
					$("#S1UeNBIP_Edit").val(data["S1UEnbIps"][0]["S1UEnbIp1"]);
				}
				if(data["S1UMmeIps"]!=undefined&&data["S1UMmeIps"]!=""){
					$("#S1UMMEIP_Edit").val(data["S1UMmeIps"][0]["S1UMmeIp1"]);
				}
				$("#CDR-Generate-IP-Edit").val(data["cdrGenerateIp"]);
				$("#CDR-Service-IP-Edit").val(data["cdrServIp"]);
				$("#CDR-Number-Edit").val(data["cdrNum"]);
				if(data["cdrGenerateIp"]!=undefined||data["cdrServIp"]!=undefined||data["cdrNum"]!=undefined||data["forwardPolicy"]=="1"){
					$("#lfpSwitch-Edit-Swtich").children().attr('isopen','true').animate({left:'24px'},100);
		    		$("#lfpSwitch-Edit-Swtich").css('background-color','#66CC66');
		    		if($(".shuntAndCreditCont").css("display")=="none"){
		    			//$(".shuntAndCreditCont").slideToggle()
		    		}
				}else{
					$("#lfpSwitch-Edit-Swtich").children().attr('isopen','false').animate({left:'1px'},100);
		            $("#lfpSwitch-Edit-Swtich").css('background-color','#d7d7d7');
		            if($(".shuntAndCreditCont").css("display")!="none"){
		    			//$(".shuntAndCreditCont").slideToggle()
		    		}
				}
				if(data["forwardPolicy"]=="1"){
					if(data["localForwardPolicys"]!=undefined&&data["localForwardPolicys"]!=""){
						for(var i=0;i<(data["localForwardPolicys"].length-1);i++){
							addLocalShuntGroup_Edit();
						}
						$('input[id="gwDNIP_Edit"]').each(function(index){
							$(this).val(data["localForwardPolicys"][index]["netIp"]);
						});
						$('input[id="gwDNM_Edit"]').each(function(index){
							$(this).val(data["localForwardPolicys"][index]["netMask"]);
						});
					}
					$("#lfpSwitch-Edit-Swtich").children().attr('isopen','true').animate({left:'24px'},100);
		    		$("#lfpSwitch-Edit-Swtich").css('background-color','#66CC66');
		    		//$(".shuntAndCreditCont").slideToggle()
				}
				if(data["forwardPolicy"]!=undefined){
					chooseShuntEdit(data["forwardPolicy"]);
				}else{
					chooseShuntEdit(2);
				}
				
				if(data["enbDynamicReg"]=="enable"){
					$("#gwDynamicreg-Edit-Swtich").children().attr('isopen','true').animate({left:'24px'},100);
		    		$("#gwDynamicreg-Edit-Swtich").css('background-color','#66CC66');
				}
				if(data["dataRelay"]=="enable"){
					$("#gwDatarelay-Edit-Swtich").children().attr('isopen','true').animate({left:'24px'},100);
		    		$("#gwDatarelay-Edit-Swtich").css('background-color','#66CC66');
				}
				if(data["sigFw"]=="enable"){
					$("#gwSigfw-Edit-Swtich").children().attr('isopen','true').animate({left:'24px'},100);
		    		$("#gwSigfw-Edit-Swtich").css('background-color','#66CC66');
				}
				if(data["upLinkSelect"]=="enable"){
					$("#gwUplinkselect-Edit-Swtich").children().attr('isopen','true').animate({left:'24px'},100);
		    		$("#gwUplinkselect-Edit-Swtich").css('background-color','#66CC66');
				}
				if(data["eGWlog"]=="enable"){
					$("#gweGWLog-Edit-Swtich").children().attr('isopen','true').animate({left:'24px'},100);
		    		$("#gweGWLog-Edit-Swtich").css('background-color','#66CC66');
				}
				if(data["opType"]=="enable"){
					$("#gwTimer-Edit-Swtich").children().attr('isopen','true').animate({left:'24px'},100);
		    		$("#gwTimer-Edit-Swtich").css('background-color','#66CC66');
		    		$("#timers-edit").val(data["timer"]);
		    		$("#timers-edit-div").show();
				}
				var param = {
						url:'${ctx}/eGW/egwManage/egwMacroCfgsList.action?TimeZone='+timeZone,
						pagination:false
		            }
		        $("#tablemacrocfglist").datagrid('load', param);	
			},
			error:function(xmlhttprequest,textstatus,errorThrown){
				$("#winLoadingPro").window("close");
				if(textstatus=="timeout"){
					$.messager.alert(TiShi, "<%=rb.getString("QingQiuChaoShi")%>");
				}else{
					$.messager.alert(TiShi, "<%=rb.getString("QingQiuYiChang")%>");
				}
			}
		});
	}
	
	function getGateWayAddInfo(gwip,gwport,gwname){
		$.ajax({
			type: "post",
			url: "${ctx}/eGW/egwManage/queryAllCfgRsp.action?TimeZone="+timeZone,
			data: {gwip:gwip,gwport:gwport,gwname:gwname},
			async: false,
			dataType:"json",
			success: function(data) {
				$("#GateWayName").val(gwname);
				$("#GateWayIP").val(gwip);
				$("#GateWayPort").val(gwport);
				$("#PLMN").val(data["plmn"]);
				if(data["enbAccessIps"]!=undefined&&data["enbAccessIps"]!=""){
					$("#eNBIp").val(data["enbAccessIps"][0]["accessIp"]);
					$("#eNBPort").val(data["enbAccessIps"][0]["accessPort"]);
				}
				if(data["S1UEnbIps"]!=undefined&&data["S1UEnbIps"]!=""){
					$("#S1UeNBIP").val(data["S1UEnbIps"][0]["S1UEnbIp1"]);
				}
				if(data["S1UMmeIps"]!=undefined&&data["S1UMmeIps"]!=""){
					$("#S1UMMEIP").val(data["S1UMmeIps"][0]["S1UMmeIp1"]);
				}
				for(var i=0;i<data["localForwardPolicys"].length;i++){
					addLocalShuntGroup();
				}
				$('input[id="gwDNIP"]').each(function(index){
					$(this).val(data["localForwardPolicys"][index]["netIp"]);
				});
				$('input[id="gwDNM"]').each(function(index){
					$(this).val(data["localForwardPolicys"][index]["netMask"]);
				});
				$("#CDR-Generate-IP").val(data["cdrGenerateIp"]);
				$("#CDR-Service-IP").val(data["cdrServIp"]);
				$("#CDR-Number").val(data["cdrNum"]);
				if(data["enbDynamicReg"]=="enable"){
					$("#gwDynamicreg-Swtich").children().attr('isopen','true').animate({left:'24px'},100);
		    		$("#gwDynamicreg-Swtich").css('background-color','#66CC66');
				}
				if(data["dataRelay"]=="enable"){
					$("#gwDatarelay-Swtich").children().attr('isopen','true').animate({left:'24px'},100);
		    		$("#gwDatarelay-Swtich").css('background-color','#66CC66');
				}
				if(data["sigFw"]=="enable"){
					$("#gwSigfw-Swtich").children().attr('isopen','true').animate({left:'24px'},100);
		    		$("#gwSigfw-Swtich").css('background-color','#66CC66');
				}
				if(data["upLinkSelect"]=="enable"){
					$("#gwUplinkselect-Swtich").children().attr('isopen','true').animate({left:'24px'},100);
		    		$("#gwUplinkselect-Swtich").css('background-color','#66CC66');
				}
				if(data["eGWlog"]=="enable"){
					$("#gweGWLog-Swtich").children().attr('isopen','true').animate({left:'24px'},100);
		    		$("#gweGWLog-Swtich").css('background-color','#66CC66');
				}
				if(data["opType"]=="enable"){
					$("#gwTimer-Swtich").children().attr('isopen','true').animate({left:'24px'},100);
		    		$("#gwTimer-Swtich").css('background-color','#66CC66');
		    		$("#timers").val(data["timer"]);
		    		$("#timers-div").show();
				}
				var param = {
						url:'${ctx}/eGW/egwManage/egwMacroCfgsList.action?TimeZone='+timeZone,
						pagination:false
		            }
		        $("#tablemacrocfglist_new").datagrid('load', param);	
			}
		});
	}
	
	//执行删除网关及宏站操作
	function delGateWayInfo(flag,gwip,gwport,gwname){
		$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingShanChuWangGuan")%>", function(r) {
	        if (r) {
	        	deleGWDataInfo(gwip);
	        	return false;
	        	/*暂时只删除记录，接口不用
	    		var root = {};
	    		root["gwName"]=gwname;
	    		root["gwIp"]=gwip;
	    		root["gwPort"]=gwport;
	    		root["tag"]="D";
	    		var json=JSON.stringify(root);
	    		$("#winLoadingPro").window("open");
	     		$.ajax({
	    			type: "post",
	    			url: "${ctx}/eGW/egwManage/setEgwJsonToXml.action?TimeZone="+timeZone,
	    			data: {jsonStr:json},
	    			async: true,
	    			dataType:"json",
	    			success: function(data) {
	    				$("#winLoadingPro").window("close");
	    				if (data["causeCode"]=="1"||data["causeCode"]==1){
	    					deleGWDataInfo(gwip);
	    					$("#tableGateWayList").datagrid("reload");
	    					$.messager.alert(TiShi, "操作成功.");
	    				}else if(data["causeCode"]=="0"||data["causeCode"]==0||data["causeCode"]==undefined){
	    					$.messager.alert(TiShi, "操作失败.");
	    				}else{
	    					$.messager.alert(TiShi, data["reasonCode"]);
	    				}
	    			}
	    		});*/
	        }
	    }).addClass("seriousConfirm");
	} 
	
	//删除网关数据
	function deleGWDataInfo(gwip){
		$.ajax({
			type: "post",
			url: "${ctx}/eGW/egwManage/deleteEgwNetConfig.action?TimeZone="+timeZone,
			data: {gwip:gwip},
			async: false,
			dataType:"json",
			success: function(data) {
				$("#tableGateWayList").datagrid("reload");
			}
		});
	}
	
	//增加网关宏站信息时列操作
	function operMacroCfgAddFormatter(value, rowData, rowIndex){
		var res="";
		res="<div class='grid-edit-btn-div' style='margin-left: 15px' title='<%=rb.getString("XiuGai")%>' onclick='newEditMacroCfgInfo(true,\"" + rowData.macroId + "\",\""+rowData.tac+"\")'></div>";
		//res="<div class='grid-del-btn-div' style='margin-left: 25px' title='<%=rb.getString("ShanChu")%>' onclick='delMacroCfgInfo(\"adddel\",\"" + rowData.macroId + "\",\""+rowData.tac+"\")'></div>";
		return res;
	}
	
	function newEditMacroCfgInfo(flag,macroId,tac){
		var gwname = $("#GateWayName").val();
		var gwip = $("#GateWayIP").val();
		var gwport = $("#GateWayPort").val();
		$("#winLoadingPro").window("open");
		$.ajax({
			type: "post",
			url: "${ctx}/eGW/egwManage/queryMacroid.action?TimeZone="+timeZone, 
			data: {gwip:gwip,gwport:gwport,macroId:macroId},
			async: true,
			dataType:"json",
			success: function(data) {
				$("#winLoadingPro").window("close");
				if(data["macroId"]==undefined||data["macroId"]==""){
					$.messager.alert(TiShi, "<%=rb.getString("QingQiuYiChang")%>");
					return false;
				}
				newMacroldSetting('newEdit');
				$("#New-MacroId").val(data["macroId"]);
				$("#New-MacroId-Hidden").val(data["macroId"]);
				$("#new-TAC").val(data["tac"]);
				$("#new-TAC-Hidden").val(data["tac"]);
				setTimeout(function(){
					$("#addMacroldTable").datagrid({
			    	   url:'${ctx}/eGW/egwManage/queryMacroidMMEList.action?TimeZone='+timeZone
			        });
				},300);
			},
			error:function(xmlhttprequest,textstatus,errorThrown){
				$("#winLoadingPro").window("close");
				if(textstatus=="timeout"){
					$.messager.alert(TiShi, "<%=rb.getString("QingQiuChaoShi")%>");
				}else{
					$.messager.alert(TiShi, "<%=rb.getString("QingQiuYiChang")%>");
				}
			}
		});
	}
	
	//修改网关宏站信息列操作列
	function operMacroCfgFormatter(value, rowData, rowIndex){
		var res="";
		res="<div class='grid-edit-btn-div' style='margin-left: 15px' title='<%=rb.getString("XiuGai")%>' onclick='editMacroCfgInfo(true,\"" + rowData.macroId + "\",\""+rowData.tac+"\")'></div>";
		//res+="<div class='grid-del-btn-div' style='margin-left: 25px' title='<%=rb.getString("ShanChu")%>' onclick='delMacroCfgInfo(\"editdel\",\"" + rowData.macroId + "\",\""+rowData.tac+"\")'></div>";
		return res;
	}
	
	//修改宏站链路信息
	function editMacroCfgInfo(flag,macroId,tac){
		$("#Edit-MacroIdCheckSpan,#Edit-TACCheckSpan").text("");
		$("#Edit-MacroId,#Edit-TAC").val("");
		$("#macrotitle").text("<%=rb.getString("XiuGaiHongZhan")%>");
		var gwname = $("#GateWayName_Edit").val();
		var gwip = $("#GateWayIP_Edit").val();
		var gwport = $("#GateWayPort_Edit").val();
		$("#winLoadingPro").window("open");
		$.ajax({
			type: "post",
			url: "${ctx}/eGW/egwManage/queryMacroid.action?TimeZone="+timeZone, 
			data: {gwip:gwip,gwport:gwport,macroId:macroId},
			async: true,
			dataType:"json",
			success: function(data) {
				$("#winLoadingPro").window("close");
				if(data["macroId"]==undefined||data["macroId"]==""){
					$.messager.alert(TiShi, "<%=rb.getString("QingQiuYiChang")%>");
					return false;
				}
				$("#query-macroid").val("");
				newMacroldSetting('edit');
				$("#Edit-GatwayName").val(gwname);
				$("#Edit-GatwayIP").val(gwip);
				$("#Edit-GatwayPort").val(gwport);
				$("#Edit-MacroId").val(data["macroId"]);
				$("#Edit-MacroId-Hidden").val(data["macroId"]);
				$("#Edit-TAC").val(data["tac"]);
				$("#Edit-TAC-Hidden").val(data["tac"]);
				setTimeout(function(){
					$("#tablemmelist").datagrid({
			    	   url:'${ctx}/eGW/egwManage/queryMacroidMMEList.action?TimeZone='+timeZone
			        });
				},300);
			},
			error:function(xmlhttprequest,textstatus,errorThrown){
				$("#winLoadingPro").window("close");
				if(textstatus=="timeout"){
					$.messager.alert(TiShi, "<%=rb.getString("QingQiuChaoShi")%>");
				}else{
					$.messager.alert(TiShi, "<%=rb.getString("QingQiuYiChang")%>");
				}
			}
		});
	}
	
	//删除宏站链路信息
	function delMacroCfgInfo(flag,macroId,tacval){
		$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingShanChuHongZhan")%>", function(r) {
			if(r){
				var mac = {};
	    		mac["tag"]="D";
	    		mac["macroId"]=macroId;
	    		var tacs=[];
	    		var tac = {};
	    		tac["tag"]="D";
	    		tac["tac"]=tacval;
	    		tacs.push(tac);
	    		mac["tacs"]=tacs;
	    		var cfg = {};
	    		cfg["macroCfg"]=mac;
	    		var root = {};
	    		var GatwayName = $("#GateWayName_Edit").val();
	    		var GatwayIP = $("#GateWayIP_Edit").val();
	    		var GatwayPort = $("#GateWayPort_Edit").val();
	    		root["gwName"]=GatwayName;
	    		root["gwIp"]=GatwayIP;
	    		root["gwPort"]=GatwayPort;
	    		var macromgmtArr=[];
	    		macromgmtArr.push(cfg);
	    		root["MacroMgmt"]=macromgmtArr;
	    		var json=JSON.stringify(root);
	    		$("#winLoadingPro").window("open");
	     		$.ajax({
	    			type: "post",
	    			url: "${ctx}/eGW/egwManage/setEgwJsonToXml.action?TimeZone="+timeZone,
	    			data: {jsonStr:json},
	    			async: true,
	    			dataType:"json",
	    			success: function(data) {
	    				$("#winLoadingPro").window("close");
	    				if (data["causeCode"]=="1"||data["causeCode"]==1){
	    					//删除宏站数据
	    					var params = {gwip: GatwayIP,macroId:macroId};
	    					$.post("${ctx}/eGW/egwManage/delMacroCfgInfo.action", params, function(data){
	    						var res=data.res;
	    					}, "json");
	    					if(flag=="adddel"){
	    						getGateWayAddInfo(GatwayIP,GatwayPort,GatwayName);
	    					}else{
	    						getGateWayEditInfo(GatwayIP,GatwayPort,GatwayName);	
	    					}
	    					closeNewMacroldSetting();
	    					//$.messager.alert(TiShi, "操作成功.");
	    				}else if(data["causeCode"]=="0"||data["causeCode"]==0||data["causeCode"]==undefined){
	    					$.messager.alert(TiShi, "<%=rb.getString("CaoZuoShiBai")%>");
	    				}else{
	    					$.messager.alert(TiShi, data["reasonCode"]);
	    				}
	    			}
	    		});
			}
	   }).addClass("seriousConfirm");	
	}
	
	//默认判断链路开关为空时显示关闭状态
	function upLinkEnableFormatter(value, rowData, rowIndex){
		if (value == null || value == "") {
			value = "disabled";
		}
		return value;
	}
	
	//链路列表操作按钮
	function operMMEFormatter(value, rowData, rowIndex){
		var res="";
		res="<div class='grid-edit-btn-div' style='margin-left: 15px' title='<%=rb.getString("XiuGai")%>' onclick='editMME(true,\"" + rowData.macroIp + "\",\""+rowData.macroPort+"\",\""+rowData.mmeIp+"\",\""+rowData.mmePort+"\",\""+rowData.upLinkEnable+"\",\""+rowIndex+"\")'></div>";
		res+="<div class='grid-del-btn-div' style='margin-left: 15px' title='<%=rb.getString("ShanChu")%>' onclick='delMME(true,\""+rowIndex+"\",\""+rowData.macroIp+"\",\""+rowData.macroPort+"\",\""+rowData.mmeIp+"\",\""+rowData.mmePort+"\",\""+rowData.upLinkEnable+"\")'></div>";
		return res;
	}
	
	//修改配置中修改链路操作
	function editMME(flag,macroIp,macroPort,mmeIp,mmePort,upLinkEnable,rowIndex){
		//验证宏站信息是否存在
		var MacroId = $("#Edit-MacroId").val();
		$('#Edit-MacroIdCheckSpan,#Edit-TACCheckSpan').text("");
		if(MacroId==""){
			$('#Edit-MacroIdCheckSpan').text("<%=rb.getString("BuNengWeiKong")%>");
			$("#Edit-MacroId").focus().select();
			return false;
		}
		var TAC = $("#Edit-TAC").val();
		if(TAC==""){
			$('#Edit-TACCheckSpan').text("<%=rb.getString("BuNengWeiKong")%>");
			$("#Edit-TAC").focus().select();
			return false;
		}
		//默认初始化显示一个ip地址框
		var GroupLength = $(".LinkInfoDefault-Edit").length;
		$(".LinkInfoDefault-Edit").find("img").each(function(index){
			if(index==1){
				subMarroIP_Edit(this);
			}
		});
		var GroupLength = $(".LinkInfommeIp-Edit").length;
		$(".LinkInfommeIp-Edit").find("img").each(function(index){
			if(index==1){
				subMarroIP_Edit(this);
			}
		});
		
		$("#macroIp-edit,#macroPort-edit,#mmeIp-edit,#mmePort-edit").val("");
		$("#opertypemme").val("edit");
		if(macroIp.indexOf("/")!=-1){
			var GroupLength = $(".LinkInfoDefault-Edit").length;
			if(GroupLength<2){
				$(".LinkInfoDefault-Edit").find("img").each(function(index){
					addMIP_Edit(this);
				});
			}
			$('input[id="macroIp-edit"]').each(function(index){
				$(this).val(macroIp.split("/")[index]);
				$("#macroIp-edit-hidden").val(macroIp.split("/")[0]);
			});
		}else{
			$("#macroIp-edit").val(macroIp);
			$("#macroIp-edit-hidden").val(macroIp);
		}
		$("#macroPort-edit").val(macroPort);
		$("#macroPort-edit-hidden").val(macroPort);
		if(mmeIp.indexOf("/")!=-1){
			var GroupLength = $(".LinkInfommeIp-Edit").length;
			if(GroupLength<2){
				$(".LinkInfommeIp-Edit").find("img").each(function(index){
					addMIP_Edit(this);
				});
			}
			$('input[id="mmeIp-edit"]').each(function(index){
				$(this).val(mmeIp.split("/")[index]);
				$("#mmeIp-edit-hidden").val(mmeIp.split("/")[0]);
			});
		}else{
			$("#mmeIp-edit").val(mmeIp);
			$("#mmeIp-edit-hidden").val(mmeIp);
		}
		$("#mmePort-edit").val(mmePort);
		$("#mmePort-edit-hidden").val(mmePort);
		if(upLinkEnable=="enable"){
			$("#upLinkEnable-Edit-Switch").children().attr('isopen','true').animate({left:'24px'},100);
    		$("#upLinkEnable-Edit-Switch").css('background-color','#66CC66');
		}else{
			$("#upLinkEnable-Edit-Switch").children().attr('isopen','false').animate({left:'1px'},100);
            $("#upLinkEnable-Edit-Switch").css('background-color','#d7d7d7');
		}
		$("#rowIndex").val(rowIndex);
		$(".addLinkInfoItem").slideToggle(250);
	}
	
	//修改配置中删除链路信息
	function delMME(flag,rowIndex,macroIp,macroPort,mmeIp,mmePort,upLinkEnable){
		$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingShanChuLianLu")%>", function(r) {
			if(r){
				var GatwayName = $("#Edit-GatwayName").val();
				var GatwayIP = $("#Edit-GatwayIP").val();
				var GatwayPort = $("#Edit-GatwayPort").val();
				var MacroId = $("#Edit-MacroId").val();
				var TAC = $("#Edit-TAC").val();
				var upLinkInfosJson={};
				var upLinkInfo={};
				var arr=[];
				var macroip="";
				upLinkInfosJson["tag"]="D";
				if(macroIp.indexOf("/")!=-1){
					macroip=macroIp.split("/")[0];
				}else{
					macroip=macroIp;
				}
				upLinkInfosJson["currentMacroIp"]=macroip;
				upLinkInfosJson["currentMacroPort"]=macroPort;
				/*
				var mmeip="";
				if(mmeIp.indexOf("/")!=-1){
					macroip=mmeIp.split("/")[0];
				}else{
					mmeip=mmeIp;
				}
				upLinkInfosJson["currentMmeIp"]=mmeip;
				upLinkInfosJson["mmePort"]=mmePort;
				
				if(upLinkEnable=="enable"){
					$("#upLinkEnable-Edit-Switch").children().attr('isopen','true').animate({left:'24px'},100);
		    		$("#upLinkEnable-Edit-Switch").css('background-color','#66CC66');
				}
				upLinkInfosJson["upLinkEnable"]=upLinkEnable;
				*/
				upLinkInfo["upLinkInfo"]=upLinkInfosJson;
				arr.push(upLinkInfo);
				var mac={};
				mac["tag"]="M";
				mac["macroId"]=MacroId;
				mac["currentMacroId"]=$("#Edit-MacroId-Hidden").val();
				var tacs={};
				tacs["tag"]="M";
				tacs["tac"]=TAC;
				var tacsarr=[];
				tacsarr.push(tacs);
				mac["tacs"]=tacsarr;
				if(arr.length>0){
					mac["upLinkInfos"]=arr;
				}
				var root = {};
				root["gwName"]=GatwayName;
				root["gwIp"]=GatwayIP;
				root["gwPort"]=GatwayPort;
				var macromgmtArr=[];
				var cfg = {};
				cfg["macroCfg"]=mac;
				macromgmtArr.push(cfg);
				root["MacroMgmt"]=macromgmtArr;
				var json=JSON.stringify(root);
				$("#winLoadingPro").window("open");
		 		$.ajax({
					type: "post",
					url: '${ctx}/eGW/egwManage/setEgwJsonToXml.action?TimeZone='+timeZone, 
					data: {jsonStr:json},
					async: true,
					dataType:"json",
					success: function(data) {
						$("#winLoadingPro").window("close");
						if (data["causeCode"]=="1"||data["causeCode"]==1){	
							editMacroCfgInfo(true,MacroId);
							cancelAddMarco();
							$.messager.alert(TiShi, "<%=rb.getString("CaoZuoChengGong")%>");
						}else if(data["causeCode"]=="0"||data["causeCode"]==0||data["causeCode"]==undefined){
							$.messager.alert(TiShi, "<%=rb.getString("CaoZuoShiBai")%>");
						}else{
							$.messager.alert(TiShi, data["reasonCode"]);
						}
					}
				});
				//$("#tablemmelist").datagrid('deleteRow',rowIndex);
			}
		}).addClass("seriousConfirm");
		
	}
	
	//添加操作链路列表操作列
	function operAddMMEFormatter(value, rowData, rowIndex){
		var res="";
		res="<div class='grid-edit-btn-div' style='margin-left: 15px' title='<%=rb.getString("FuZhi")%>' onclick='editMME_Add(true,\"" + rowData.macroIp + "\",\""+rowData.macroPort+"\",\""+rowData.mmeIp+"\",\""+rowData.mmePort+"\",\""+rowData.upLinkEnable+"\",\""+rowIndex+"\")'></div>";
		res+="<div class='grid-del-btn-div' style='margin-left: 15px' title='<%=rb.getString("FuZhi")%>' onclick='delMME_Add(true,\""+rowIndex+"\",\""+rowData.macroIp+"\",\""+rowData.macroPort+"\",\""+rowData.mmeIp+"\",\""+rowData.mmePort+"\",\""+rowData.upLinkEnable+"\")'></div>";
		return res;
	}
	
	//增加操作时对MME链路进行修改操作
	function editMME_Add(flag,macroIp,macroPort,mmeIp,mmePort,upLinkEnable,rowIndex){
		//校验宏站信息是否存在
		$('#New-MacroIdCheckSpan,#new-TACCheckSpan').text("");
		if($("#New-MacroId").val()==""){
			$('#New-MacroIdCheckSpan').text("<%=rb.getString("BuNengWeiKong")%>");
			$("#New-MacroId").focus().select();
			return false;
		}
		if($("#new-TAC").val()==""){
			$('#new-TACCheckSpan').text("<%=rb.getString("BuNengWeiKong")%>");
			$("#new-TAC").focus().select();
			return false;
		}
		
		$("#macroIp-add,#macroPort-add,#mmeIp-add,#mmePort-add").val("");
		$("#opertypemme-add").val("edit");
		if(macroIp.indexOf("/")!=-1){
			var GroupLength = $(".LinkInfoDefault").length;
			if(GroupLength<2){
				$(".LinkInfoDefault").find("img").each(function(index){
					addMIP(this);
				});
			}
			$('input[id="macroIp-add"]').each(function(index){
				$(this).val(macroIp.split("/")[index]);
				$("#macroIp-add-hidden").val(macroIp.split("/")[0]);
			});
		}else{
			$("#macroIp-add").val(macroIp);
			$("#macroIp-add-hidden").val(macroIp);
		}
		$("#macroPort-add").val(macroPort);
		$("#macroPort-add-hidden").val(macroPort);
		if(mmeIp.indexOf("/")!=-1){
			var GroupLength = $(".LinkInfommeIp").length;
			if(GroupLength<2){
				$(".LinkInfommeIp").find("img").each(function(index){
					addMIP(this);
				});
			}
			$('input[id="mmeIp-add"]').each(function(index){
				$(this).val(mmeIp.split("/")[index]);
				$("#mmeIp-add-hidden").val(mmeIp.split("/")[0]);
			});
		}else{
			$("#mmeIp-add").val(mmeIp);
			$("#mmeIp-add-hidden").val(mmeIp);
		}
		$("#mmePort-add").val(mmePort);
		$("#mmePort-add-hidden").val(mmePort);
		if(upLinkEnable=="enable"){
			$("#upLinkEnable-Add-Switch").children().attr('isopen','true').animate({left:'24px'},100);
    		$("#upLinkEnable-Add-Switch").css('background-color','#66CC66');
		}else{
			$("#upLinkEnable-Add-Switch").children().attr('isopen','false').animate({left:'1px'},100);
            $("#upLinkEnable-Add-Switch").css('background-color','#d7d7d7');
		}
		$("#rowIndex-add").val(rowIndex);
		$(".addLinkInfoItem").slideToggle(250);
	}
	
	//删除链路信息
	function delMME_Add(flag,rowIndex,macroIp,macroPort,mmeIp,mmePort,upLinkEnable){
		$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingShanChuLianLu")%>", function(r) {
			if(r){
				var GatwayName = $("#New-GatwayName").val();
				var GatwayIP = $("#New-GatwayIP").val();
				var GatwayPort = $("#New-GatwayPort").val();
				var MacroId = $("#New-MacroId").val();
				var oldMacroId = $("#New-MacroId-Hidden").val();
				var TAC = $("#new-TAC").val();
				var oldTac = $("#new-TAC-Hidden").val();
				var upLinkInfosJson={};
				var upLinkInfo={};
				var arr=[];
				var macroip="";
				upLinkInfosJson["tag"]="D";
				if(macroIp.indexOf("/")!=-1){
					macroip=macroIp.split("/")[0];
				}else{
					macroip=macroIp;
				}
				upLinkInfosJson["currentMacroIp"]=macroip;
				upLinkInfosJson["currentMacroPort"]=macroPort;
				upLinkInfo["upLinkInfo"]=upLinkInfosJson;
				arr.push(upLinkInfo);
				var mac={};
				mac["tag"]="M";
				mac["macroId"]=MacroId;
				mac["currentMacroId"]=oldMacroId;
				var tacs={};
				tacs["tag"]="M";
				tacs["tac"]=TAC;
				var tacsarr=[];
				tacsarr.push(tacs);
				mac["tacs"]=tacsarr;
				if(arr.length>0){
					mac["upLinkInfos"]=arr;
				}
				var root = {};
				root["gwName"]=GatwayName;
				root["gwIp"]=GatwayIP;
				root["gwPort"]=GatwayPort;
				var macromgmtArr=[];
				var cfg = {};
				cfg["macroCfg"]=mac;
				macromgmtArr.push(cfg);
				root["MacroMgmt"]=macromgmtArr;
				var json=JSON.stringify(root);
		 		$.ajax({
					type: "post",
					url: '${ctx}/eGW/egwManage/setEgwJsonToXml.action?TimeZone='+timeZone, 
					data: {jsonStr:json},
					async: false,
					dataType:"json",
					success: function(data) {
						if (data["causeCode"]=="1"||data["causeCode"]==1){
							editMacroCfgInfo(true,MacroId);
							cancelAddMarco();
							$.messager.alert(TiShi, "<%=rb.getString("CaoZuoChengGong")%>");
						}else if(data["causeCode"]=="0"||data["causeCode"]==0){
							$.messager.alert(TiShi, "<%=rb.getString("CaoZuoShiBai")%>");
						}else{
							$.messager.alert(TiShi, data["reasonCode"]);
						}
					}
				});
				//$("#addMacroldTable").datagrid('deleteRow',rowIndex);
			}
		}).addClass("seriousConfirm");
	}
	
	//修改链路信息
	var macroIpJson={};
	function editLinkInfo(){
		var arr=[];
		var n=1;
		macroIpJson={};
		var macroIp="";
		var flag=false;
		$('input[id="macroIp-edit"]').each(function(i){
			if(n==2){
				macroIp+="/"
			}
			$('span[id="'+this.id+'CheckSpan"]').text("");
			if(this.value!=""){
				if(!isValidIP(this.value)){
					$('span[id="'+this.id+'CheckSpan"]').each(function(j){
						if(j==i){
							$(this).text("<%=rb.getString("IPGeShiBuDui")%>");
							return false;
						}
					});
					$(this).focus().select();
					flag=true;
					return false;
				}else{
					macroIp+=macroIpJson["macroIp"+i]=this.value;
					n++;
				}
			}else{
				$('span[id="'+this.id+'CheckSpan"]').text("");
				$('span[id="'+this.id+'CheckSpan"]').each(function(j){
					if(i==j){
						$(this).text("<%=rb.getString("BuNengWeiKong")%>");	
					}
				});
				$(this).focus().select();
				flag=true;
				return false;
			}
		});
		if(flag){
			return false;
		}
		var mmeIp="";
		n=1;
		$('input[id="mmeIp-edit"]').each(function(i){
			if(n==2){
				mmeIp+="/"
			}
			$('span[id="'+this.id+'CheckSpan"]').text("");
			if(this.value!=""){
				if(!isValidIP(this.value)){
					$('span[id="'+this.id+'CheckSpan"]').each(function(j){
						if(j==i){
							$(this).text("<%=rb.getString("IPGeShiBuDui")%>");
							return false;
						}
					});
					$(this).focus().select();
					flag=true;
					return false;
				}else{
					mmeIp+=macroIpJson["mmeIp"+i]=this.value;
					n++;
				}
			}else{
				$('span[id="'+this.id+'CheckSpan"]').text("");
				$('span[id="'+this.id+'CheckSpan"]').each(function(j){
					if(i==j){
						$(this).text("<%=rb.getString("BuNengWeiKong")%>");	
					}
				});
				$(this).focus().select();
				flag=true;
				return false;
			}
		});
		if(flag){
			return false;
		}
		var macroPort = $("#macroPort-edit").val();
		if(macroPort==""){
			$('#macroPort-editCheckSpan').text("<%=rb.getString("BuNengWeiKong")%>");
			$("#macroPort-edit").focus().select();
			return false;
		}
		if(!checkPort("macroPort-edit",macroPort)){
			return false;
		}
		var mmePort = $("#mmePort-edit").val();
		if(mmePort==""){
			$('#mmePort-editCheckSpan').text("<%=rb.getString("BuNengWeiKong")%>");
			$("#mmePort-edit").focus().select();
			return false;
		}
		if(!checkPort("mmePort-edit",mmePort)){
			return false;
		}
		var rowIndex = $("#rowIndex").val();
		var upLinkEnable=$("#upLinkEnable-Edit").attr('isopen') == 'true'?"enable":"disable";
		var opertype = $("#opertypemme").val();
		/*
		if(opertype == 'edit'){
			$("#tablemmelist").datagrid('updateRow',{index:rowIndex,row:{macroIp:macroIp,macroPort:macroPort,mmeIp:mmeIp,mmePort:mmePort,upLinkEnable:upLinkEnable}});
		}else if(opertype == 'add'){
			var datarows = $("#tablemmelist").datagrid('getData');
			$("#tablemmelist").datagrid('insertRow',{index:datarows.rows.length+1,row:{macroIp:macroIp,macroPort:macroPort,mmeIp:mmeIp,mmePort:mmePort,upLinkEnable:upLinkEnable}});
		}*/
		//////////////////////////////////////////////
		var operSaveType = $("#operSaveType").val();
		var upLinkInfosJson={};
		var arr=[];
		//验证链路信息
		var datarows = $("#tablemmelist").datagrid('getData');
		for(var i=0;i<datarows.rows.length;i++){
			var macroIprow = datarows.rows[i].macroIp;
			var mmeIprow = datarows.rows[i].mmeIp;
			if(macroIprow.indexOf("/")!=-1){
				macroIprow=datarows.rows[i].macroIp.split("/")[0];
			}
			if(mmeIprow.indexOf("/")!=-1){
				mmeIprow=datarows.rows[i].mmeIp.split("/")[0];
			}
			var newmacroip = macroIp;
			if(newmacroip.indexOf("/")!=-1){
				newmacroip=newmacroip.split("/")[0];
			}
			var newmmeip = mmeIp;
			if(newmmeip.indexOf("/")!=-1){
				newmmeip=newmmeip.split("/")[0];
			}
			if(macroIprow==newmacroip&&mmeIprow==newmmeip&&newmacroip!=$("#macroIp-edit-hidden").val()){
				flag=true;
				break;
			}
		}
		if(flag){
			$("#macroIp-editCheckSpan").text("<%=rb.getString("YiCunZai")%>");
			return false;
		}
		var upLinkInfoJson={};
		var upLinkInfo={};
		//只要是修改宏站，添加链路均为修改状态
		if(opertype == 'edit'){
			upLinkInfoJson["tag"]="M";
			upLinkInfoJson["currentMacroIp"]=$("#macroIp-edit-hidden").val();
			upLinkInfoJson["currentMacroPort"]=$("#macroPort-edit-hidden").val();
			upLinkInfoJson["currentMmeIp"]=$("#mmeIp-edit-hidden").val();
			upLinkInfoJson["currentMmePort"]=$("#mmePort-edit-hidden").val();
		}else if(opertype == 'add'){
			upLinkInfoJson["tag"]="A";
		}

		upLinkInfoJson["macroIp"]=macroIp.replace("/","|");
		upLinkInfoJson["macroPort"]=macroPort;
		upLinkInfoJson["mmeIp"]=mmeIp.replace("/","|");
		upLinkInfoJson["mmePort"]=mmePort;
		upLinkInfoJson["upLinkEnable"]=upLinkEnable;
		upLinkInfo["upLinkInfo"]=upLinkInfoJson;
		arr.push(upLinkInfo);
		
		var GatwayName = $("#Edit-GatwayName").val();
		var GatwayIP = $("#Edit-GatwayIP").val();
		var GatwayPort = $("#Edit-GatwayPort").val();
		var MacroId = $("#Edit-MacroId").val();
		var TAC = $("#Edit-TAC").val();
		
		var mac = {};
		
		if(operSaveType=="ADD"){
			mac["tag"]="A";
		}else{
			mac["tag"]="M";
			mac["currentMacroId"]=$("#Edit-MacroId-Hidden").val();
		}
		mac["macroId"]=MacroId;
		var tacs=[];
		var tac = {};
		if(operSaveType=="ADD"){
			tac["tag"]="A";
		}else{
			tac["tag"]="M";
		}
		tac["tac"]=TAC;
		//tac["currentTac"]=$("#Edit-TAC-Hidden").val();
		tacs.push(tac);
		mac["tacs"]=tacs;
		
		if(arr.length>0){
			mac["upLinkInfos"]=arr;
		}
		var root = {};
		root["gwName"]=GatwayName;
		root["gwIp"]=GatwayIP;
		root["gwPort"]=GatwayPort;
		var macromgmtArr=[];
		var cfg = {};
		cfg["macroCfg"]=mac;
		macromgmtArr.push(cfg);
		root["MacroMgmt"]=macromgmtArr;
		var json=JSON.stringify(root);

		$("#winLoadingPro").window("open");
 		$.ajax({
			type: "post",
			url: '${ctx}/eGW/egwManage/setEgwJsonToXml.action?TimeZone='+timeZone, 
			data: {jsonStr:json,macroId:MacroId,tac:TAC,oldmacroId:$("#Edit-MacroId-Hidden").val()},
			async: true,
			dataType:"json",
			success: function(data) {
				$("#winLoadingPro").window("close");
				if (data["causeCode"]=="1"||data["causeCode"]==1){
					var params = {gwip:GatwayIP,gwport:GatwayPort,gwname:GatwayName}
					$.post("${ctx}/eGW/egwManage/queryAllCfgRsp.action?TimeZone="+timeZone, params, function(data){
						$("#tablemacrocfglist").datagrid({
							url:'${ctx}/eGW/egwManage/egwMacroCfgsList.action?TimeZone='+timeZone,pagination:false
				        });
					}, "json");
					editMacroCfgInfo(true,MacroId);
					cancelAddMarco();
					//$.messager.alert(TiShi, "操作成功.");
				}else if(data["causeCode"]=="0"||data["causeCode"]==0||data["causeCode"]==undefined){
					$.messager.alert(TiShi, "<%=rb.getString("CaoZuoShiBai")%>");
				}else{
					$.messager.alert(TiShi, data["reasonCode"]);
				}
			}
		});
	}
	
	//修改宏站信息
	function editMacroConfiguration(){
		var operSaveType = $("#operSaveType").val();
		var upLinkInfosJson={};
		/*var arr=[];
		var datarows = $("#tablemmelist").datagrid('getData');
		for(var i=0;i<datarows.rows.length;i++){
			var upLinkInfoJson={};
			var upLinkInfo={};
			if(operSaveType=="ADD"){
				upLinkInfoJson["tag"]="A";
			}else{
				upLinkInfoJson["tag"]="M";
			}
			var macroIp = datarows.rows[i].macroIp;
			upLinkInfoJson["macroIp"]=macroIp.replace("/","|");
			upLinkInfoJson["macroPort"]=datarows.rows[i].macroPort;
			var mmeIp = datarows.rows[i].mmeIp;
			upLinkInfoJson["mmeIp"]=mmeIp.replace("/","|");
			upLinkInfoJson["mmePort"]=datarows.rows[i].mmePort;
			upLinkInfoJson["upLinkEnable"]=datarows.rows[i].upLinkEnable;
			upLinkInfo["upLinkInfo"]=upLinkInfoJson;
			arr.push(upLinkInfo);
		}
		if(arr.length>0){
			upLinkInfosJson["upLinkInfos"]=arr;
		}*/
		var GatwayName = $("#Edit-GatwayName").val();
		var GatwayIP = $("#Edit-GatwayIP").val();
		var GatwayPort = $("#Edit-GatwayPort").val();
		var MacroId = $("#Edit-MacroId").val();
		$('#Edit-MacroIdCheckSpan,#Edit-TACCheckSpan').text("");
		if(MacroId==""){
			$('#Edit-MacroIdCheckSpan').text("<%=rb.getString("BuNengWeiKong")%>");
			$("#Edit-MacroId").focus().select();
			return false;
		}
		var TAC = $("#Edit-TAC").val();
		if(TAC==""){
			$('#Edit-TACCheckSpan').text("<%=rb.getString("BuNengWeiKong")%>");
			$("#Edit-TAC").focus().select();
			return false;
		}
		
		var mac = {};
		if(operSaveType=="ADD"){
			mac["tag"]="A";
		}else{
			mac["tag"]="M";
			mac["currentMacroId"]=$("#Edit-MacroId-Hidden").val();
		}
		mac["macroId"]=MacroId;
		var tacs=[];
		var tac = {};
		if(operSaveType=="ADD"){
			tac["tag"]="A";
		}else{
			tac["tag"]="M";
		}
		tac["tac"]=TAC;
		tacs.push(tac);
		mac["tacs"]=tacs;
		var cfg = {};
		cfg["macroCfg"]=mac;
		var root = {};
		root["gwName"]=GatwayName;
		root["gwIp"]=GatwayIP;
		root["gwPort"]=GatwayPort;
		var macromgmtArr=[];
		//macromgmtArr.push(cfg);
		/*
		if(!$.isEmptyObject(upLinkInfosJson)){
			cfg["macroCfg"]=upLinkInfosJson;
		}
		*/
		if(!$.isEmptyObject(cfg)){
			macromgmtArr.push(cfg);
		}
		root["MacroMgmt"]=macromgmtArr;
		var json=JSON.stringify(root);
		$("#winLoadingPro").window("open");
 		$.ajax({
			type: "post",
			url: '${ctx}/eGW/egwManage/setEgwJsonToXml.action?TimeZone='+timeZone, 
			data: {jsonStr:json,macroId:MacroId,tac:TAC,oldmacroId:$("#Edit-MacroId-Hidden").val()},
			async: true,
			dataType:"json",
			success: function(data) {
				$("#winLoadingPro").window("close");
				$("#query-macroid").val("");
				if (data["causeCode"]=="1"||data["causeCode"]==1){
					var params = {gwip:GatwayIP,gwport:GatwayPort,gwname:GatwayName}
					$.post("${ctx}/eGW/egwManage/queryAllCfgRsp.action?TimeZone="+timeZone, params, function(data){
						$("#tablemacrocfglist").datagrid({
							url:'${ctx}/eGW/egwManage/egwMacroCfgsList.action?TimeZone='+timeZone,pagination:false
				        });
					}, "json");
					closeNewMacroldSetting();
					//$.messager.alert(TiShi, "操作成功.");
				}else if(data["causeCode"]=="0"||data["causeCode"]==0||data["causeCode"]==undefined){
					$.messager.alert(TiShi, "<%=rb.getString("CaoZuoShiBai")%>");
				}else{
					$.messager.alert(TiShi, data["reasonCode"]);
				}
			}
		});
	}
	
	function addLinkInfoTo(){
		var arr=[];
		var n=1;
		macroIpJson={};
		var macroIp="";
		var flag=false;
		$('input[id="macroIp-add"]').each(function(i){
			if(n==2){
				macroIp+="/"
			}
			$('span[id="'+this.id+'CheckSpan"]').text("");
			if(this.value!=""){
				if(!isValidIP(this.value)){
					$('span[id="'+this.id+'CheckSpan"]').each(function(j){
						if(j==i){
							$(this).text("<%=rb.getString("IPGeShiBuDui")%>");
							return false;
						}
					});
					$(this).focus().select();
					flag=true;
					return false;
				}else{
					macroIp+=macroIpJson["macroIp"+i]=this.value;
					n++;
				}
			}else{
				$('span[id="'+this.id+'CheckSpan"]').each(function(j){
					if(i==j){
						$(this).text("<%=rb.getString("BuNengWeiKong")%>");	
					}
				});
				$(this).focus().select();
				flag=true;
				return false;
			}
		});
		if(flag){
			return false;
		}
		var mmeIp="";
		n=1;
		$('input[id="mmeIp-add"]').each(function(i){
			if(n==2){
				mmeIp+="/"
			}
			$('span[id="'+this.id+'CheckSpan"]').text("");
			if(this.value!=""){
				if(!isValidIP(this.value)){
					$('span[id="'+this.id+'CheckSpan"]').each(function(j){
						if(j==i){
							$(this).text("<%=rb.getString("IPGeShiBuDui")%>");
							return false;
						}
					});
					$(this).focus().select();
					flag=true;
					return false;
				}else{
					mmeIp+=macroIpJson["mmeIp"+i]=this.value;
					n++;
				}
			}else{
				$('span[id="'+this.id+'CheckSpan"]').each(function(j){
					if(i==j){
						$(this).text("<%=rb.getString("BuNengWeiKong")%>");	
					}
				});
				$(this).focus().select();
				flag=true;
				return false;
			}
		});
		if(flag){
			return false;
		}
		
		var macroPort = $("#macroPort-add").val();
		if(macroPort==""){
			$('#macroPort-addCheckSpan').text("<%=rb.getString("BuNengWeiKong")%>");
			$("#macroPort-add").focus().select();
			return false;
		}
		if(!checkPort("macroPort-add",macroPort)){
			return false;
		}
		var mmePort = $("#mmePort-add").val();
		if(mmePort==""){
			$('#mmePort-addCheckSpan').text("<%=rb.getString("BuNengWeiKong")%>");
			$("#mmePort-add").focus().select();
			return false;
		}
		if(!checkPort("mmePort-add",mmePort)){
			return false;
		}
		var rowIndex = $("#rowIndex-add").val();
		var upLinkEnable=$("#upLinkEnable-Add").attr('isopen') == 'true'?"enable":"disable";
		var opertype = $("#opertypemme-add").val();
		
		/*
		if(opertype == 'edit'){
			$("#addMacroldTable").datagrid('updateRow',{index:rowIndex,row:{macroIp:macroIp,macroPort:macroPort,mmeIp:mmeIp,mmePort:mmePort,upLinkEnable:upLinkEnable}});
		}else if(opertype == 'add'){
			var datarows = $("#tablemmelist").datagrid('getData');
			$("#addMacroldTable").datagrid('insertRow',{index:datarows.rows.length+1,row:{macroIp:macroIp,macroPort:macroPort,mmeIp:mmeIp,mmePort:mmePort,upLinkEnable:upLinkEnable}});
		}
		*/
		/////////////////////////
		
		//验证链路信息
		var datarows = $("#addMacroldTable").datagrid('getData');
		for(var i=0;i<datarows.rows.length;i++){
			var macroIprow = datarows.rows[i].macroIp;
			var mmeIprow = datarows.rows[i].mmeIp;
			if(macroIprow.indexOf("/")!=-1){
				macroIprow=datarows.rows[i].macroIp.split("/")[0];
			}
			if(mmeIprow.indexOf("/")!=-1){
				mmeIprow=datarows.rows[i].mmeIp.split("/")[0];
			}
			var newmacroip = macroIp;
			if(newmacroip.indexOf("/")!=-1){
				newmacroip=newmacroip.split("/")[0];
			}
			var newmmeip = mmeIp;
			if(newmmeip.indexOf("/")!=-1){
				newmmeip=newmmeip.split("/")[0];
			}
			if(macroIprow==newmacroip&&mmeIprow==newmmeip&&newmacroip!=$("#macroIp-add-hidden").val()){
				flag=true;
				break;
			}
		}
		if(flag){
			$("#macroIp-addCheckSpan").text("<%=rb.getString("YiCunZai")%>");
			$("#macroIp-add").focus().select();
			return false;
		}
		
		var macro = {};
		var tacs=[];
		var tac={};
		var operSaveTypeadd = $("#operSaveType-add").val();
		if(operSaveTypeadd=="ADD"){
			macro["tag"]="A";
			tac["tag"]="A";
		}else{
			macro["tag"]="M";
			tac["tag"]="M";
			macro["currentMacroId"]=$("#New-MacroId-Hidden").val();
		}
		macro["macroId"]=$("#New-MacroId").val();
		
		tac["tac"]=$("#new-TAC").val();
		
		tacs[0]=tac;
		macro["tacs"]=tacs;
		var uli=[];
		var upLinkInfo={};
		var uplink={};
		if(opertype == 'edit'){
			upLinkInfo["tag"]="M";
			upLinkInfo["currentMacroIp"]=$("#macroIp-add-hidden").val();
			upLinkInfo["currentMacroPort"]=$("#macroPort-add-hidden").val();
			upLinkInfo["currentMmeIp"]=$("#mmeIp-add-hidden").val();
			upLinkInfo["currentMmePort"]=$("#mmePort-add-hidden").val();
		}else if(opertype == 'add'){
			upLinkInfo["tag"]="A";
		}
		upLinkInfo["macroIp"]=macroIp.replace("/","|");
		upLinkInfo["macroPort"]=macroPort;
		upLinkInfo["mmeIp"]=mmeIp.replace("/","|");
		upLinkInfo["mmePort"]=mmePort;
		upLinkInfo["upLinkEnable"]=upLinkEnable;
		uplink["upLinkInfo"]=upLinkInfo;
		uli[0]=uplink;
		if(uli.length>0){
			macro["upLinkInfos"]=uli;
		}
		var cfg = {};
		cfg["macroCfg"]=macro;
		var mgmt=[];
		mgmt[0]=cfg;
		
		var root = {};
		
		root["gwName"]=$("#New-GatwayName").val();
		root["gwIp"]=$("#New-GatwayIP").val();
		root["gwPort"]=$("#GateWayPort").val();
		root["MacroMgmt"]=mgmt;
		
		var json=JSON.stringify(root);
		$("#winLoadingPro").window("open");
 		$.ajax({
			type: "post",
			url: "${ctx}/eGW/egwManage/setEgwJsonToXml.action?TimeZone="+timeZone, 
			data: {jsonStr:json,tac:$("#new-TAC").val(),macroId:$("#New-MacroId").val(),oldmacroId:$("#New-MacroId-Hidden").val()},
			async: true,
			dataType:"json",
			success: function(data) {
				$("#winLoadingPro").window("close");
				if (data["causeCode"]=="1"||data["causeCode"]==1){
					var params = {gwip:$("#New-GatwayIP").val(),gwport:$("#GateWayPort").val(),gwname:$("#New-GatwayName").val()}
					$.post("${ctx}/eGW/egwManage/queryAllCfgRsp.action?TimeZone="+timeZone, params, function(data){
						$("#tablemacrocfglist_new").datagrid({
							url:'${ctx}/eGW/egwManage/egwMacroCfgsList.action?TimeZone='+timeZone,pagination:false
				        });
					}, "json");
					setTimeout(function(){
						$("#addMacroldTable").datagrid({
				    	   url:'${ctx}/eGW/egwManage/queryMacroidMMEList.action?TimeZone='+timeZone
				        });
					},300);
					cancelAddMarco();
					closeNewMacroldSetting();
				}else if(data["causeCode"]=="0"||data["causeCode"]==0||data["causeCode"]==undefined){
					$.messager.alert(TiShi, "<%=rb.getString("CaoZuoShiBai")%>");
				}else{
					$.messager.alert(TiShi, data["reasonCode"]);
				}
			}
		}); 
 		
	}
	
	function addMacroConfiguration(){
		var macro = {};
		macro["tag"]="A";
		$('#New-MacroIdCheckSpan,#new-TACCheckSpan').text("");
		if($("#New-MacroId").val()==""){
			$('#New-MacroIdCheckSpan').text("<%=rb.getString("BuNengWeiKong")%>");
			$("#New-MacroId").focus().select();
			return false;
		}
		macro["macroId"]=$("#New-MacroId").val();
		
		var tacs=[];
		var tac={};
		tac["tag"]="A";
		if($("#new-TAC").val()==""){
			$('#new-TACCheckSpan').text("<%=rb.getString("BuNengWeiKong")%>");
			$("#new-TAC").focus().select();
			return false;
		}
		tac["tac"]=$("#new-TAC").val();
		
		tacs[0]=tac;
		macro["tacs"]=tacs;
		/*
		var uli=[];
		var datarows = $("#addMacroldTable").datagrid('getData');
		for(var i=0;i<datarows.rows.length;i++){
			var upLinkInfo={};
			var uplink={};
			upLinkInfo["tag"]="A";
			var macroIp = datarows.rows[i].macroIp;
			upLinkInfo["macroIp"]=macroIp.replace("/","|");
			upLinkInfo["macroPort"]=datarows.rows[i].macroPort;
			var mmeIp = datarows.rows[i].mmeIp;
			upLinkInfo["mmeIp"]=mmeIp.replace("/","|");
			upLinkInfo["mmePort"]=datarows.rows[i].mmePort;
			upLinkInfo["upLinkEnable"]=datarows.rows[i].upLinkEnable;
			uplink["upLinkInfo"]=upLinkInfo;
			uli[i]=uplink;
		}
		if(uli.length>0){
			macro["upLinkInfos"]=uli;
		}
		*/
		var cfg = {};
		cfg["macroCfg"]=macro;
		var mgmt=[];
		mgmt[0]=cfg;
		
		var root = {};
		
		root["gwName"]=$("#New-GatwayName").val();
		root["gwIp"]=$("#New-GatwayIP").val();
		root["gwPort"]=$("#GateWayPort").val();
		root["MacroMgmt"]=mgmt;
		
		var json=JSON.stringify(root);
		$("#winLoadingPro").window("open");
 		$.ajax({
			type: "post",
			url: "${ctx}/eGW/egwManage/setEgwJsonToXml.action?TimeZone="+timeZone, 
			data: {jsonStr:json,macroId:macro["macroId"],tac:tac["tac"]},
			async: true,
			dataType:"json",
			success: function(data) {
				$("#winLoadingPro").window("close");
				if (data["causeCode"]=="1"||data["causeCode"]==1){
					var params = {gwip:root["gwIp"],gwport:root["gwPort"],gwname:root["gwName"]}
					$.post("${ctx}/eGW/egwManage/queryAllCfgRsp.action?TimeZone="+timeZone, params, function(data){
						$("#tablemacrocfglist_new").datagrid({
							url:'${ctx}/eGW/egwManage/egwMacroCfgsList.action?TimeZone='+timeZone,pagination:false
				        });
					}, "json");
					closeNewMacroldSetting();
					//$.messager.alert(TiShi, "操作成功.");
				}else if(data["causeCode"]=="0"||data["causeCode"]==0||data["causeCode"]==undefined){
					$.messager.alert(TiShi, "<%=rb.getString("CaoZuoShiBai")%>");
				}else{
					$.messager.alert(TiShi, data["reasonCode"]);
				}
			}
		}); 
	}
	
	function addLinkInfos(){

		var macroid=$("#New-MacroId").val();
		var tac=$("#new-TAC").val();
		
		var macroip="";
		$("input[name='add-macro-ip']").each(function(index,element){
			if(index>0){
				macroip=macroip+"|"+$(this).val();
			}else{
				macroip=macroip+$(this).val();
			}
		});
		var macroport=$("#add-macro-port").val();
		var mmeip="";
		$("input[name='add-mme-ip']").each(function(index,element){
			if(index>0){
				mmeip=mmeip+"|"+$(this).val();
			}else{
				mmeip=mmeip+$(this).val();
			}
		});
		var mmeprot=$("#add-mme-port").val();
		var link_switch;
		if ($("#add-uplink-switch").attr('isopen') == 'false'){
			link_switch="<div class='switch' style='background-color:#d7d7d7'><div isopen='false' class='btnn' style='left:2px;'></div></div>"
		}else{
			link_switch="<div class='switch' style='background-color:#66CC66'><div isopen='true' class='btnn' style='left:2px;'></div></div>"
		}
		var oper="<div class='grid-edit-btn-div' title='' style='margin-left:15px;' onclick=''></div><div class='grid-del-btn-div' title='' style='margin-left:15px;' onclick=''></div>";
		var trhtml="<tr><td>1</td><td>"+macroip+"</td><td>"+macroport+"</td><td>"+mmeip+"</td><td>"+mmeprot+"</td><td>"+link_switch+"<td>"+oper+"</td>";
		$("#addMacroList").append(trhtml);
		$(".addLinkInfoItem").slideUp(300);
		//初始化页面元素
		$(".addLinkInfoItem input").val("");
		$("#uls").children().attr('isopen','false');
        $("#uls").css('background-color','#d7d7d7');
        if($(".subMacroIP").length>0){
        	$(".subMacroIP").click();
        }
	}
	
	//点击查询获取指定宏站数据列
	function getMacroidInfo(){
		var macroId = $("#query-macroid").val();
		var gwIp = $("#GateWayIP_Edit").val();
		if(gwIp==""){
			gwIp = $("#GateWayIP").val();
		}
		$("#tablemacrocfglist").datagrid({
	    	   url:'${ctx}/eGW/egwManage/getEewNetConfigMacroidList.action?TimeZone='+timeZone,
	    	   pagination:true,
	    	   queryParams:{
	    		   macroId : macroId,
	    		   gwIp : gwIp
	    	   }
	    });
	}
	
	function isValidIP(ip){     
	    var reg =  /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/     
	    return reg.test(ip);     
	}    

	function checkIpfun(id,value){  
	    if(!isValidIP(value)&&value.length!=0){
	    	$("#"+id).focus().select();
	        $("#"+id+"CheckSpan,#"+id+"CheckSpan-Edit").text("<%=rb.getString("IPGeShiBuDui")%>");
	        return false;
	    }else{
	    	$("#"+id+"CheckSpan,#"+id+"CheckSpan-Edit").text("");
	    	return true;
	    }
	}
	
	//校验端口号 范围 0-65535
	function checkPort(id,value){
		if(isNumeric(value)&& parseInt(value)>=0 && parseInt(value)<=65535){
			$("#"+id+"CheckSpan,#"+id+"CheckSpan").text("");
			return true;
		}else{
			$("#"+id).focus().select();
	        $("#"+id+"CheckSpan,#"+id+"CheckSpan").text("<%=rb.getString("DuanKouGeShiBuDui")%>(0~65535)");
	        return false;
		}
	}
	
	function isNumeric(str) {
	    if(str.length==0){
	    	return false;
	    }
	    for(var i=0;i<str.length;i++){
	    	if(str.charAt(i)<"0" || str.charAt(i)>"9"){
	    		return false;
	    	}
	    }
	    return true;  
	} 
	
	//校验子网掩码
	function checkMask(markIp){
		var exp=/^(254|252|248|240|224|192|128|0)\.0\.0\.0|255\.(254|252|248|240|224|192|128|0)\.0\.0|255\.255\.(254|252|248|240|224|192|128|0)\.0|255\.255\.255\.(254|252|248|240|224|192|128|0)$/; 
		return exp.test(markIp); 
		/*
		if(exp.test(e.value)){
			$("#"+idx).text("");
		}else{
			$("#"+idx).text("mark format error.");
			e.value="";
			e.focus();
		}*/
	}
	
	function checkPlmn(id,value){
		if(value.length<5||value.length>6){
			$("#"+id).focus().select();
	        $("#"+id+"CheckSpan,#"+id+"CheckSpan").text("<%=rb.getString("ChangDu5Dao6WeiCunShuZi")%>");
			return true;
		}else{
			$("#"+id+"CheckSpan,#"+id+"CheckSpan").text("");
	        return false;
		}
	}
	
	//清空面板
	function emptyAll(){
		$("input[type='text']").val("");
		$(".prompt").text("");
		$(".switch").children().attr('isopen','false').animate({left:'2px'},100);
        $(".switch").css('background-color','#d7d7d7');
        $(".shuntAndCreditCont").show();
        //$("input[name='shuntType'][value='2']").attr("checked",true);
		//$("#tablemacrocfglist_new,#addMacroldTable,#tablemacrocfglist,#tablemmelist").datagrid("loadData",{total:0,rows:[]});
	}
</script>