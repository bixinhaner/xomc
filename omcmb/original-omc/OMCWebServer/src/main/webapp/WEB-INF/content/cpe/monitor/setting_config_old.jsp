<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>
<style>
	.el-card__body{
		display:flex;
		flex-direction:column;
		flex:1 1 auto;
		height:100%;
		overflow:auto;
	}
</style>
<div style='display:flex;flex-direction:column;height:100%;'>
	<div class='el-card__header'>
		<span><%=rb.getString("SheZhi")%></span>
		<span class='el-icon el-icon-close' onclick='closeCpeSettingOption()'></span>
	</div>
	<div class="el-card__body" id="cpe_setting_ctn">
		<div class="flex-item" style='background:#fff;'>
			<div style='margin-left:20px;margin-top:20px;'>
			<!-- 基本信息 -->
		       <div id="basicInfoDiv" style="margin-bottom:10px;" class='form-group'>
					<div class="group-title not-extend">
						<span class="title-icon"></span>
						<span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
					</div>
		   			<div class="qosOptionDetailsLeft" style="width:500px;margin-bottom:30px;">					
						<label class="inputTittleCss"><%=rb.getString("MingCheng")%>:</label>
						<input class="inputDivCss border border-box" oldValue="" id="CPE_NAME" name="CPE_NAME" onblur="cpe_validateHostName(event)" 
							max_length="45" title=""/>
						<label class="inputTipCss" id="cpeNameTitle"><%=rb.getString("SheBeiMingChengGuiZe")%></label>
					</div>    		
		       </div>
		       <!-- pci lock -->
		       <div id='pciLockDiv' class='form-group' style='margin-bottom:10px;'>
		       		<div class="group-title not-extend">
						<span class="title-icon"></span>
						<span class="title-text"><%=rb.getString("SuoPin")%></span>
					</div>
					<div class='qosOptionDetailsLeft' style='margin-bottom:30px;width:auto'>
						<div style='margin-bottom:20px;'>
							<label><%=rb.getString("SaoMiaoFangShi")%></label>
							<select id="scanMode" name="CPE_scanMode" class="easyui-combobox border border-box item" data-options="editable:false,onSelect:selectPciLock" style="height:26px;width:300px;">
								<option value="fullband">Full Band</option>
								<option id="cpeFreqTypeOpt" value="freqpreferred">Band/Frequency Preferred</option>
								<option value="pcilock">PCI lock</option>
								<option value="pcionlylock">PCI Only Lock</option>
							</select>
						</div>
						<!-- 添加频点 -->
						<div style='display:none' class='frequencyItem'>
							<div style='height:75px;width:350px;'>
								<label><%=rb.getString("PinDian")%></label>
								<input id="addFrequencyInput" onblur="validatePciLock(this,'frequency')" name="cpe_frequency" min_val = 0 max_val=65535 title='0 - 65535' class="border border-box"/>
								<span onclick='addEarfcn(this)' style='margin-left:10px;vertical-align:middle' class='el-icon el-icon-plus'></span>
								<p class='msg_error'></p>
							</div>
							<div id='earfcnOptions'></div>
						</div>
						<!-- 添加频点和PCI -->
						<div style='display:none' class='pciLockItem'>
							<div style='height:75px;'>
								<p style='display:inline-block'>
									<label><%=rb.getString("PinDian")%></label>
									<input id="addEarfcnInput" onblur="validatePciLock(this,'pciLock')" name="cpe_earfcn" min_val = 0 max_val=65535 title='0 - 65535' class="border border-box"/>
								</p>
								<span style='margin-left:5px;margin-right:5px;'>:</span>
								<p  style='display:inline-block'>
									<label><%=rb.getString("PCIZhi")%></label>
									<input id="addPciInput" onblur="validatePciLock(this,'pciLock')" name="cpe_pci" min_val = 0 max_val=503 title='0 - 503' class="border border-box"/>
								</p>
								<span onclick='addEarfcnAndPci()' style='margin-left:10px;vertical-align:middle' class='el-icon el-icon-plus'></span>
								<p class='msg_error'></p>
							</div>
							<div id='earfcnAndPciOptions'></div>
						</div>
						<!-- 添加PCI Only Lock -->
						<div style='display:none' class='pcionlylockItem'>
							<div style='height:75px;width:350px;'>
								<label>PCI</label>
								<input id="addPciOnlyLockInput" onblur="validatePciLock(this,'pciOnlyLock')" name="cpe_pci" min_val = 0 max_val=503 title='0 - 503' class="border border-box" style='width:300px;'/>
								<span onclick='addPciOnlyLock(this)' style='margin-left:10px;vertical-align:middle' class='el-icon el-icon-plus'></span>
								<p class='msg_error'></p>
							</div>
							<div id='pciOnlyLockOptions'></div>
						</div>
					</div>
		       </div>
		       	<!-- 远程WEB设置 -->
		       <div id="remoteWebDiv" style="margin-bottom:10px;" class='form-group'> 
					<div class="group-title not-extend">
						<span class="title-icon"></span>
						<span class="title-text"><%=rb.getString("WanKouDengLuSheZhi")%></span>
					</div>
		       		<div class='qosOptionDetailsLeft' style='width:auto;margin-bottom:30px;'>
		       			<div class='httpItem' style='display:inline-block'>
		       				<label><%=rb.getString("WanKouDengLu")%></label>
			       			<div style='vertical-align:sub;float:none;margin-right:90px;' class='switch' onclick="openWebFunction(this)">
								<div id='httpSwitch' isopen='true' value='1' oldValue="1" class='btnn fnSwitch' style='left:24px;'></div>
							</div>
		       			</div>
		       			<!-- 暂时不开放LAN接口，勿删 -->
		       			<div id="laninterface_div" class='lanItem' style='display:inline-block'>
		       				<label>LAN <%=rb.getString("JieKou")%></label>
		       				<div style='vertical-align:sub;float:none' class='switch' onclick="openWebFunction(this)">
								<div id='lanSwitch' isopen='true' value='1' oldValue="1" class='btnn fnSwitch' style='left:24px;'></div>
							</div>
		       			</div>
						
		       		</div>
		       </div>
		       <!-- DMZ设置 -->
		       <div id="dmzSettingDiv" style="margin-bottom:10px;display: none;" class='form-group'> 
					<div class="group-title not-extend">
						<span class="title-icon"></span>
						<span class="title-text">DMZ <%=rb.getString("SheZhi")%></span>
					</div>
		       		<div class='qosOptionDetailsLeft' style='width:auto;margin-bottom:30px;'>
	       				<label>DMZ <%=rb.getString("1588KaiGuan")%></label>
		       			<div style='vertical-align:sub;float:none;margin-right:90px;' class='switch' onclick="openDMZFunction(this)">
							<div id='dmzSwitch' isopen='true' value='1' oldValue="1" class='btnn fnSwitch' style='left:24px;'></div>
						</div>
		       		</div>
		       </div>
		       <!-- Watchdog设置 -->
		       <div id="watchdogSettingDiv" style="margin-bottom:10px;" class='form-group'> 
					<div class="group-title not-extend">
						<span class="title-icon"></span>
						<span class="title-text">Ping Watchdog <%=rb.getString("SheZhi")%></span>
					</div>
		       		<div class='qosOptionDetailsLeft' style='width:auto;margin-bottom:30px;'>
	       				<label>Ping Watchdog <%=rb.getString("1588KaiGuan")%></label>
		       			<div style='vertical-align:sub;float:none;margin-right:90px;' class='switch' onclick="openDMZFunction(this)">
							<div id='watchdogSwitch' isopen='true' value='1' oldValue="1" class='btnn fnSwitch' style='left:24px;'></div>
						</div>
						
						<div style='margin-bottom:20px;margin-top:20px;'>
							<label class="inputTittleCss">IP Address or URL to Ping:</label>
							<input class="inputDivCss border border-box" oldValue="" id="IP_Address" name="watchDogPingIp" onblur="watchdog_validateIP(event)" 
								maxlength="45" title=""/>
							<label class="inputTipCss" id="IP_Address_tip"><%=rb.getString("SheBeiMingChengGuiZe")%></label>
						</div>
						
						<div style='margin-bottom:20px;'>
							<label class="inputTittleCss">Ping Timeout(Seconds):</label>
							<input class="inputDivCss border border-box" oldValue="" id="Ping_Timeout" name="watchDogPingTimeout" onblur="watchdog_validate(event,this)" 
								maxlength="8" title=""/>
							<label class="inputTipCss" id="Ping_Timeout_tip">Range: 1 - 65535</label>
						</div>
						
						<div style='margin-bottom:20px;'>
							<label class="inputTittleCss">Ping Count:</label>
							<input class="inputDivCss border border-box" oldValue="" id="Ping_Count" name="watchDogPingCount" onblur="watchdog_validate(event,this)" 
								maxlength="8" title=""/>
							<label class="inputTipCss" id="Ping_Count_tip">Range: 1 - 65535</label>
						</div>
						
						<div style='margin-bottom:20px;'>
							<label class="inputTittleCss">Failure Count to Reboot:</label>
							<input class="inputDivCss border border-box" oldValue="" id="Faulure_Reboot" name="watchDogFailureReboot" onblur="watchdog_validate(event,this)" 
								maxlength="8" title=""/>
							<label class="inputTipCss" id="Faulure_Reboot_tip">Range: 1 - 65535</label>
						</div>
		       		</div>
		       </div>
			   <!-- AP LIST 列表-->
			   <div id="APListDiv" style="margin-bottom:10px;position: relative;" > 
					<div class="group-title not-extend">
						<span class="title-icon"></span>
						<span class="title-text">AP LIST</span>
						<div class="circleIcon placeholder-bt" tip="<%=rb.getString("ShuaXin")%>" style="top: 0px;right: 20px;" onclick="refreshAPList()">
							<span slot="reference" class="el-icon-circle-refresh el-icon"></span>
						</div>
					</div>
		       		<div  style='width:auto;margin:20px 30px 10px 20px;'>
		       			<table id="APListDatagrid"></table>
		       		</div>
		       </div>
			</div>
		</div>
	</div>
	<div id="operateDiv" class='slideFooter'>
			<div id="operateDivDetails">
				<a class="linkbutton linkbutton_trend" href="javascript:void(0)" onclick="cpeSettingCommit()"><span><%=rb.getString("QueDing")%></span></a>
				<a class="linkbutton linkbutton_nowanna" href="javascript:void(0)" onclick="closeCpeSettingOption()"><span><%=rb.getString("QuXiao")%></span></a>
			</div>
	</div>
</div>
<script>
	var frequencyCount = 1,
		pciLockCount = 1,
		pcionlylockCount = 1,
		suopinDisableFlag,
		cpeCode = "${cpeCode}",
		product = "${product}",
		softVersion = "${softVersion}",
		cpeName = "${cpeName}",
		scanMode,
		isDMZEnable = false;
	var oldFreArr = [],old_earfcn_arr = [],old_pci_arr = [],old_pcionlylock_arr=[];
	var httpsParamFlag;

	var lanEnableFlag = sessionStorage.getItem('lanFlag'),
		isLANEnable = [0,1,'0','1'].includes(lanEnableFlag);
	
	$(function(){
		closeLoading();
		var reg = new RegExp("^(IDU\/CN)");
		var regversion = new RegExp("^(?!BCE-IDU)");
		if ((product == "LTE WiFi VoIP Gateway" || (reg.test(product)==true)) && (regversion.test(softVersion)== true) ) {
			//禁用
			suopinDisableFlag = true;
			$("#pciLockDiv").hide();
	    } else {
	    	//可用
	    	if(writableMap["CODE_CPE_PCI_LOCK"]){
	    		suopinDisableFlag = false;
		    	$("#pciLockDiv").show();
	    	}else{
	    		suopinDisableFlag = true;
				$("#pciLockDiv").hide();
	    	}
	    }
		
		if(!isLANEnable) {
			$('#laninterface_div').hide();
		}
		
		if(!isLWAEnable) {
			$('#APListDiv').hide();
		}else {
			//APList任务列表初始化加载
			$("#APListDatagrid").datagrid({
				//APList表格请求接口
				url : '${ctx}/cell/ap/queryCPEAPInfos.action',
				border:false,
				rownumbers:true,
				fitColumns:true,
				height:180,
				pagination:true,
				pagePosition:'bottom',
				striped:true,
				singleSelect:true,
				// idField:'serial_number',
				queryParams: {cpeCode:cpeCode},
				columns:[[
						// { field:'serial_number',hidden:true},
						{ field:'serial_number',title:'Serial Number',width:30},
						{ field:'ssid',title:'SSID',width:150,},
						{ field:'encryption',title:'Encryption',width:80}, 
						{ field:'key',title:'Key',width:100},
					]],
			})
		}

		
		$.post("${ctx}/cell/CPE/getSettingParams.action",{cpeCode:cpeCode},function(data){
			var cpeFreqType = data["cpeFreqType"];
			var frequency = data["frequency"];
			var pci = data["PCI"];
			var PCI_value = data["PCI_value"];
			scanMode = data["lockMode"];
			if(scanMode == undefined){
				scanMode = '';
			}
			if(cpeFreqType=="ODUZM" || cpeFreqType=="BAIYI"){
				$("#cpeFreqTypeOpt").show();
			}else{
				$("#cpeFreqTypeOpt").hide();
			}
			$("#scanMode").combobox("setValue",scanMode);
			if(scanMode == "fullband"){
				$(".frequencyItem").hide();
				$(".pciLockItem").hide();
			}else if(scanMode == "freqpreferred"){
				$(".frequencyItem").show();
				$(".pciLockItem").hide();
				$('.pcionlylockItem').hide();
				if(frequency != undefined && frequency.indexOf(',') >= 0){
					var frequencyArr = frequency.split(',');
					frequencyArr.map(function(item,index){
						if(index == 0){
							$("#addFrequencyInput").val(item);
							$("#addFrequencyInput").attr("oldValue",item);
						}else{
							var str = '<div style="height:75px;"><label><%=rb.getString("PinDian")%></label><input value='+item+' oldValue='+item+' onblur="validatePciLock(this,\'frequency\')" name="cpe_pci" min_val = 0 max_val=503 title="0 - 503" class="border border-box"/>';
							str += '<span onclick="deleteEarfcn(this)" style="margin-left:10px;vertical-align:middle" class="el-icon el-icon-minus"></span>'
							str += '<p class="msg_error"></p></div>'
							$("#earfcnOptions").append(str)
							frequencyCount ++;
						}
					})
				}else{
					$("#addFrequencyInput").val(frequency);
					$("#addFrequencyInput").attr("oldValue",frequency);
				}
				$(".frequencyItem input.border").map(function(index,item){
					oldFreArr.push($(item).attr("oldValue"));
				})
			}else if(scanMode == "pcilock"){
				$(".frequencyItem").hide();
				$(".pciLockItem").show();
				$('.pcionlylockItem').hide();
				if(frequency != undefined && frequency.indexOf(",") >= 0){
					var earfcnArr = frequency.split(",");
					var pciArr = pci.split(",");
					earfcnArr.map(function(item,index){
						if(index == 0){
							$("#addEarfcnInput").val(item);
							$("#addEarfcnInput").attr("oldValue",item);
							$("#addPciInput").val(pciArr[0]);
							$("#addPciInput").attr("oldValue",pciArr[0]);
						}else{
							var str = '<div style="height:75px;"><p style="display:inline-block"><label><%=rb.getString("PinDian")%></label><input value='+item+' oldValue='+item+' onblur="validatePciLock(this,\'pciLock\')" name="cpe_earfcn" min_val = 0 max_val=65535 title="0 - 65535" class="border border-box"/></p>';
							str += '<span style="margin-left:5px;margin-right:5px;"> : </span>';
							str += '<p style="display:inline-block"><label>PCI</label><input value='+pciArr[index]+' oldValue='+pciArr[index]+' onblur="validatePciLock(this,\'pciLock\')" name="cpe_pci" min_val = 0 max_val=503 title="0 - 503" class="border border-box"/></p>';
							str += '<span onclick="deleteEarfcnAndPci(this)" style="margin-left:10px;vertical-align:middle" class="el-icon el-icon-minus"></span>';
							str += '<p class="msg_error"></p></div>'
							$("#earfcnAndPciOptions").append(str);
							pciLockCount ++;
						}
					})
				}else{
					$("#addEarfcnInput").val(frequency);
					$("#addEarfcnInput").attr("oldValue",frequency);
					$("#addPciInput").val(pci);
					$("#addPciInput").attr("oldValue",pci);
				}
				$(".pciLockItem input[name=cpe_earfcn]").map(function(index,item){
					old_earfcn_arr.push($(item).attr("oldValue"));
				})
				$(".pciLockItem input[name=cpe_pci]").map(function(index,item){
					old_pci_arr.push($(item).attr("oldValue"));
				})
			}else if(scanMode == "pcionlylock"){
				$('.pcionlylockItem').show();
				$(".frequencyItem").hide();
				$(".pciLockItem").hide();
				if(PCI_value != undefined && PCI_value.indexOf(',') >= 0){
					var pciValueArr = PCI_value.split(',');
					pciValueArr.map(function(item,index){
						if(index == 0){
							$("#addPciOnlyLockInput").val(item);
							$("#addPciOnlyLockInput").attr("oldValue",item);
						}else{
							var str = '<div style="height:75px;"><label>PCI</label><input value='+item+' oldValue='+item+' onblur="validatePciLock(this,\'pciOnlyLock\')" name="cpe_pci" min_val = 0 max_val=65535 title="0 - 503" class="border border-box" style="width:300px;"/>';
							str += '<span onclick="deletePciOnlyLock(this)" style="margin-left:10px;vertical-align:middle" class="el-icon el-icon-minus"></span>'
							str += '<p class="msg_error"></p></div>'
							$("#pciOnlyLockOptions").append(str)
							pcionlylockCount ++;
						}
					})
				}else{
					$("#addPciOnlyLockInput").val(PCI_value);
					$("#addPciOnlyLockInput").attr("oldValue",PCI_value);
				}
				$(".frequencyItem input.border").map(function(index,item){
					oldFreArr.push($(item).attr("oldValue"));
				})
			}
			
			if( data["httpsFlag"] == "1" || data["httpsFlag"] == "0" ){
				httpsParamFlag = false;
			}else if ( data["httpsFlag"] == "True" || data["httpsFlag"] == "False"){
				httpsParamFlag = true;
			} 
			
			
			if( data["httpsFlag"] == "1" || data["httpsFlag"] == "True" ){
				 $("#httpSwitch").attr("value","1");
				 $("#httpSwitch").attr("oldValue","1");
			}else if ( data["httpsFlag"] == "0" || data["httpsFlag"] == "False"){
				openWebFunction($("#httpSwitch").parent());
				$("#httpSwitch").attr("value","0");
				$("#httpSwitch").attr("oldValue","0");
			} 
			
			if(data["dmzEnable"] == "1" || data["dmzEnable"] == "0") {
				$('#dmzSettingDiv').show();
				isDMZEnable = true;
			}
			
			if(isDMZEnable) {
				// DMZ 回显
				if( data["dmzEnable"] == "1" ){
					 $("#dmzSwitch").attr("value","1");
					 $("#dmzSwitch").attr("oldValue","1");
				}else {
					openDMZFunction($("#dmzSwitch").parent());
					$("#dmzSwitch").attr("value","0");
					$("#dmzSwitch").attr("oldValue","0");
				} 
			}
			
			if(isLANEnable) {
				if("1" == data["lanFlag"]){
					$("#lanSwitch").attr("value","1");
					$("#lanSwitch").attr("oldValue","1");
				}else{
					openWebFunction($("#lanSwitch").parent());
					$("#lanSwitch").attr("value","0");
					$("#lanSwitch").attr("oldValue","0");
				}
			}
			
			if("1" == data["lanFlag"]){
				$("#watchdogSwitch").attr("value","1");
				$("#watchdogSwitch").attr("oldValue","1");
			}else{
				openWebFunction($("#lanSwitch").parent());
				$("#watchdogSwitch").attr("value","0");
				$("#watchdogSwitch").attr("oldValue","0");
			}
			
			if("0"== data["connFlag"]){
				 $(".httpItem").addClass("forbidden");
				 //$(".lanItem").addClass("forbidden");
				 $("#CPE_NAME").prop("disabled",false);
			} else {
				$(".httpItem").removeClass("forbidden");
				//$(".lanItem").removeClass("forbidden");
				$("#CPE_NAME").prop("disabled",false);
			}
			
			initInputs(document.querySelector('#cpe_setting_ctn'));
		},"json")
		$("#CPE_NAME").val(cpeName);
		$("#CPE_NAME").attr("oldValue", cpeName==null?'':cpeName);
	})
	/* 添加频点 */
	function addEarfcn(ele){
		if(frequencyCount == 3){
			return false
		}else{
			var str = '<div style="height:75px;"><label><%=rb.getString("PinDian")%></label><input onblur="validatePciLock(this,\'frequency\')" name="cpe_frequency" min_val = 0 max_val=65535 title="0 - 65535" class="border border-box"/>';
			str += '<span onclick="deleteEarfcn(this)" style="margin-left:10px;vertical-align:middle" class="el-icon el-icon-minus"></span>'
			str += '<p class="msg_error"></p></div>'
			$("#earfcnOptions").append(str)
			frequencyCount ++;
		}
	}
	/* 删除频点 */
	function deleteEarfcn(ele){
		$(ele).parent()[0].remove();
		frequencyCount --;
	}
	/* 添加频点和PCI */
	function addEarfcnAndPci(){
		if(pciLockCount == 3){
			return false
		}else{
			var str = '<div style="height:75px;"><p style="display:inline-block"><label><%=rb.getString("PinDian")%></label><input onblur="validatePciLock(this,\'pciLock\')" name="cpe_earfcn" min_val = 0 max_val=65535 title="0 - 65535" class="border border-box"/></p>';
			str += '<span style="margin-left:5px;margin-right:5px;"> : </span>';
			str += '<p style="display:inline-block"><label>PCI</label><input onblur="validatePciLock(this,\'pciLock\')" name="cpe_pci" min_val = 0 max_val=503 title="0 - 503" class="border border-box"/></p>';
			str += '<span onclick="deleteEarfcnAndPci(this)" style="margin-left:10px;vertical-align:middle" class="el-icon el-icon-minus"></span>';
			str += '<p class="msg_error"></p></div>'
			$("#earfcnAndPciOptions").append(str);
			pciLockCount ++;
		}
	}
	/* 删除频点和PCI */
	function deleteEarfcnAndPci(ele){
		$(ele).parent()[0].remove();
		pciLockCount --;
	}
	/* 添加PCI Only Lock */
	function addPciOnlyLock(ele){
		if(pcionlylockCount == 3){
			return false
		}else{
			var str = '<div style="height:75px;"><label>PCI</label><input onblur="validatePciLock(this,\'pciOnlyLock\')" name="cpe_pci" min_val = 0 max_val=503 title="0 - 503" class="border border-box" style="width:300px;"/>';
			str += '<span onclick="deletePciOnlyLock(this)" style="margin-left:10px;vertical-align:middle" class="el-icon el-icon-minus"></span>'
			str += '<p class="msg_error"></p></div>'
			$("#pciOnlyLockOptions").append(str)
			pcionlylockCount ++;
		}
	}
	function deletePciOnlyLock(ele){
		$(ele).parent()[0].remove();
		pcionlylockCount --;
	}
	/* pci lock下拉框改变 */
	function selectPciLock(record){
		var val = record.value;
		if(val == 'fullband'){
			$(".frequencyItem").hide();
			$(".pciLockItem").hide();
			$(".pcionlylockItem").hide();
		}else if(val == 'freqpreferred'){
			$(".frequencyItem").show();
			$(".pciLockItem").hide();
			$(".pcionlylockItem").hide();
		}else if(val == 'pcilock'){
			$(".frequencyItem").hide();
			$(".pciLockItem").show();
			$(".pcionlylockItem").hide();
		}else if(val == 'pcionlylock'){
			$('.pcionlylockItem').show();
			$(".frequencyItem").hide();
			$(".pciLockItem").hide();
		}
	}
	/* 校验频点和pci */
	function validatePciLock(ele,type){
		var target,
		value = $(ele).val(),
		minVal = parseInt($(ele).attr("min_val")),
		maxVal = parseInt($(ele).attr("max_val")),
		reg = /^(0|[1-9][0-9]*)$/,
		name = $(ele).attr("name");
		if(type == 'frequency' || type == 'pciOnlyLock'){
			target = $(ele).siblings('p.msg_error');
		}else if(type == 'pciLock'){
			target = $(ele).parent().siblings('p.msg_error');
		}
		if(name == 'cpe_earfcn' || name == 'cpe_frequency'){
			if(value == ""){
				target.html("<%=rb.getString("PinDianWeiKong")%>").show();
				$(ele).addClass("border_error");
			}else if(!reg.test(value)){
				target.html("<%=rb.getString("PinDianGeShiCuoWu")%>").show();
				$(ele).addClass("border_error");
			}else if(value < minVal || value > maxVal){
				target.html("<%=rb.getString("PinDianChaoChuFanWei")%>").show();
				$(ele).addClass("border_error");
			}else{
				target.html("").hide();
				$(ele).removeClass("border_error");
			}
		}else if(name == 'cpe_pci'){
			if(value == ""){
				target.html("<%=rb.getString("PCIWeiKong")%>").show();
				$(ele).addClass("border_error");
			}else if(!reg.test(value)){
				target.html("<%=rb.getString("PCIGeShiCuoWu")%>").show();
				$(ele).addClass("border_error");
			}else if(value < minVal || value > maxVal){
				target.html("<%=rb.getString("PCIChaoChuFanWei")%>").show();
				$(ele).addClass("border_error");
			}else{
				target.html("").hide();
				$(ele).removeClass("border_error");
			}
		}
	}
	/* 开关打开和关闭 */
	function openWebFunction(ele){
		if ($(ele).children().attr('isopen') == 'false') {
			$(ele).children().attr('isopen','true').animate({left:'24px'},100);
			$(ele).css('background-color','#66CC66');
			$(ele).children().attr("value","1");
		} else {
			$(ele).children().attr('isopen','false').animate({left:'1px'},100);
	        $(ele).css('background-color','#838383');
	        $(ele).children().attr("value","0");
		}
	}
	/* DMZ开关打开和关闭 */
	function openDMZFunction(ele){
		if ($(ele).children().attr('isopen') == 'false') {
			$(ele).children().attr('isopen','true').animate({left:'24px'},100);
			$(ele).css('background-color','#66CC66');
			$(ele).children().attr("value","1");
		} else {
			$(ele).children().attr('isopen','false').animate({left:'1px'},100);
	        $(ele).css('background-color','#838383');
	        $(ele).children().attr("value","0");
		}
	}
	/* 刷新AP列表 */
	function refreshAPList() {
		$.post("${ctx}/cell/ap/refreshCPEApList.action", {cpeCode: cpeCode}, function(data){
	 		if(data["success"]){
	 			
	 		}else{
	 			showMsg('error_msg',data["message"]);
	 		}
		}, "json");
	}
	/* cpe名称校验 */
	function cpe_validateHostName(e){
		var ele = $(e["target"]);
	    var currVal = ele.val();
	    var reg = /^\s\S{0,45}$/;
	    if(currVal.length>45){
	    	$("#" + ele.attr("id") + "_err").show();
	        ele.addClass("err_border");
		   /*  if(!reg.test(currVal)){
		    	$("#" + ele.attr("id") + "_err").show();
		        ele.addClass("err_border");
		    }else{
		    	 $("#" + ele.attr("id") + "_err").hide();
		         ele.removeClass("err_border");
		    } */
	    }else{
	         ele.removeClass("err_border");
	    }
	}
	/* watchdog ip校验 */
	function watchdog_validateIP(e){
		var ele = $(e["target"]);
	    var currVal = ele.val(),
    		enable = $("#watchdogSwitch").attr("value");
	    
	    if(currVal.length>45){
	    	$("#" + ele.attr("id") + "_err").show();
	        ele.addClass("err_border");
	    }else{
	    	if(!currVal && enable == 1) {
	    		$("#" + ele.attr("id") + "_tip").show();
		        ele.addClass("err_border");
	    	}else {
	    		ele.removeClass("err_border");
	    	}
	    }
	}
	/* watchdog相关校验 */
	function watchdog_validate(e){
		var ele = $(e["target"]);
	    var currVal = ele.val(),
	    	enable = $("#watchdogSwitch").attr("value");
	    
	    if(currVal) {
		    if(currVal-1<0 || currVal-65535>0){
		    	$("#" + ele.attr("id") + "_tip").show();
		        ele.addClass("err_border");
		    }else{
		        ele.removeClass("err_border");
		    }
	    }else {
	    	if(enable == 1) {
	    		$("#" + ele.attr("id") + "_tip").show();
		        ele.addClass("err_border");
	    	}else {
	    		ele.removeClass("err_border");
	    	}
	    }
	}
	/*设置完成提交*/	
	var alertFlag = false;
	function cpeSettingCommit(){
		var nameChangeFlag = false,
			freChangeFlag = false,
			earfcnChangeFlag = false,
			pciChangeFlag = false,
			pcionlylockChangeFlag = false,
			httpChangeFlag = false,
			lanChangeFlag = false,
			dmzChangeFlag = false,
			codeChangeFlag = false,
			rightPci = false;
		var mode = $("#scanMode").combobox("getValue"),
			cpeName = $("#CPE_NAME").val(),
			oldCpeName = $("#CPE_NAME").attr("oldValue"),
			http = $("#httpSwitch").attr("value"),
			oldHttp = $("#httpSwitch").attr("oldValue"),
			freArr = [],earfcn_arr = [],pci_arr = [],pcionlylock_arr = [],
			lan = $("#lanSwitch").attr("value"),
			oldLan = $("#lanSwitch").attr("oldValue")
			dmz = $("#dmzSwitch").attr("value"),
			oldDmz = $("#dmzSwitch").attr("oldValue");
		
		var watchdogEnable = $("#watchdogSwitch").attr("value"),
			oldwatchdog = $("#dmzSwitch").attr("oldValue"),
			watchdogIp = $('#IP_Address').val(),
			watchdogTimeout = $('#Ping_Timeout').val(),
			watchdogCount = $('#Ping_Count').val(),
			watchdogFailure = $('#Faulure_Reboot').val();
			
		var params = {
				cpeCode : cpeCode,
				timeZone : timeZone
			};
		
		if(watchdogEnable == oldwatchdog) {
			params.watchDogEnable = watchdogEnable;
		}
		
		if(suopinDisableFlag == false){
				if(scanMode == mode){
					codeChangeFlag = true;
				}else{
					params.lockMode = mode;
				}
				if(mode == 'freqpreferred'){
					$(".frequencyItem input.border").map(function(index,item){
						$(item).blur();
						if($(item).hasClass("border_error")){
							rightPci = true;
						}
						freArr.push($(item).val());
					})
					if(isRepeat(freArr)){
						showMsg("prompt_msg","<%=rb.getString("YouChongFuShuJu")%>");
					}
					if(freArr.toString() == oldFreArr.toString()){
						freChangeFlag = true;
					}else{
						params.lockMode = mode;
						params.CPE_Frequency = freArr.toString();
					}
				}else if(mode == 'pcilock'){
					$(".pciLockItem input[name=cpe_earfcn]").map(function(index,item){
						$(item).blur();
						if($(item).hasClass("border_error")){
							rightPci = true;
						}
						earfcn_arr.push($(item).val());
					})
					$(".pciLockItem input[name=cpe_pci]").map(function(index,item){
						$(item).blur();
						if($(item).hasClass("border_error")){
							rightPci = true;
						}
						pci_arr.push($(item).val())
					})
					if(isRepeat(earfcn_arr) && isRepeat(pci_arr)){
						showMsg("prompt_msg","<%=rb.getString("YouChongFuShuJu")%>");
						return;
					}
					if(earfcn_arr.toString() == old_earfcn_arr.toString()){
						earfcnChangeFlag = true;
					}
					if(pci_arr.toString() == old_pci_arr.toString()){
						pciChangeFlag = true;
					}
					if(earfcnChangeFlag && pciChangeFlag){//均无改变
						
					}else{
						params.lockMode = mode;
						params.CPE_Frequency = earfcn_arr.toString();
						params.PCI_value = pci_arr.toString();
					}
				}else if(mode == 'pcionlylock'){
					$(".pcionlylockItem input.border").map(function(index,item){
						$(item).blur();
						if($(item).hasClass("border_error")){
							rightPci = true;
						}
						pcionlylock_arr.push($(item).val());
					})
					if(isRepeat(pcionlylock_arr)){
						showMsg("prompt_msg","<%=rb.getString("YouChongFuShuJu")%>");
					}
					if(pcionlylock_arr.toString() == old_pcionlylock_arr.toString()){
						pcionlylockChangeFlag = true;
					}else{
						params.lockMode = mode;
						params.PCI_value = pcionlylock_arr.toString().split(",").join(";");
					}
				}
		}
		if(rightPci){
			return;
		}
		if(oldCpeName == cpeName){
			nameChangeFlag = true;
			params.cpeNameChanged = 0;
		}else{
			params.cpeName = cpeName;
			params.cpeNameChanged = 1;
		}
		if(http == oldHttp){
			httpChangeFlag = true;
		}else{
			if ( httpsParamFlag ){
				http = http == "0" ? "False" : "True";
			}
			params.httpsEnable = http;
		}
		
		if(oldDmz == dmz){
			dmzChangeFlag = true;
		}else{
			params.dmzEnable = dmz;
		}
		
		if(isLANEnable) {
			if(oldLan == lan){
				lanChangeFlag = true;
			}else{
				params.lanEnable = lan;
			}
		}
		var boolUnit = isLANEnable? lanChangeFlag:true,
			dmzBool = isDMZEnable? dmzChangeFlag:true,
			isInputChanged = isValueChanged(document.querySelector('#watchdogSettingDiv'));
		
		if(isInputChanged) {
			var formMap = getChangedValue(document.querySelector('#watchdogSettingDiv'));
			Object.assign(params,formMap);
		}
		
		if(suopinDisableFlag == true){
			if(nameChangeFlag && httpChangeFlag && boolUnit && dmzBool && isInputChanged){
				showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
				return;
			}
		}else{
			if(mode == 'fullband' || mode == ''){
				if(nameChangeFlag && codeChangeFlag && httpChangeFlag && boolUnit && dmzBool && isInputChanged){
					showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
					return;
				}
			}else if(mode == 'freqpreferred'){
				if(freChangeFlag && nameChangeFlag && codeChangeFlag && httpChangeFlag && boolUnit && dmzBool && isInputChanged){
					showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
					return;
				}
			}else if(mode == 'pcilock'){
				if(earfcnChangeFlag && pciChangeFlag && nameChangeFlag && codeChangeFlag && httpChangeFlag && boolUnit && dmzBool && isInputChanged){
					showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
					return;
				}
			}else if(mode == 'pcionlylock'){
				if(pcionlylockChangeFlag && nameChangeFlag && codeChangeFlag && httpChangeFlag && boolUnit && dmzBool && isInputChanged){
					showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
					return;
				}
			}
		}
		progressDivShow();
		$("#cpeSettingOption").addClass("loading");
		$.post("${ctx}/cell/CPE/setCpeParams.action", params, function(data){
	 		if(data["success"]){
	 			progressDivHide();
	 			closeCpeSettingOption();
	 			$("#cpeSettingOption").removeClass("loading");
	 			$("#tableHomeCpeList").datagrid("reload");
	 		}else{
	 			showMsg('error_msg',data["message"]);
	 		}
		}, "json");
	}
	function closeCpeSettingOption(){
		$("#cpeSettingOption").animate({right:'-1700px'},500,function(){
			$("#cpeSettingOption").html("");
		});
	}
	function isRepeat(arr){
		var hash = {};
		for(var i in arr){
			if(hash[arr[i]]){
				return true;
			}
			hash[arr[i]] = true;
		}
		return false;
	}
</script>