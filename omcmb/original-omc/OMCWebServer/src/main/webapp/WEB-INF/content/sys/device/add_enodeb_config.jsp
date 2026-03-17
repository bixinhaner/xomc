<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
#addEnb_header {
	position: absolute;
	width: 100%;
	line-height: 50px;
	height: 50px;
	top: 0px;
	color: #7993B6;
	font-size: 16px;
}
#addEnb_body {
	/*position: absolute;*/
	width: 100%;
	top: 50px;
	overflow: auto;
	bottom: 10px;
}
#enbInfo .errorTipStyle{
	visibility:hidden;
}
.forbidden > span{
    background: #B0CBDD !important;
}
.commonHidden, .enbaleCommon{ display: none; }
.optionDetailsLeft, .optionDetailsRight { margin-top: 0; display: inline-block; }

.errorTipStyle { margin-top: -4px; }
</style>

<div id="addEnb_body" style="height: 470px;" class="flex-ctn">
	<div style="margin-bottom:10px;" id="enbInfo">
		<div class="optionDetailsLeft">					
			<label class="inputTittleCss"><span style="color: red;">*</span> <%=rb.getString("XiaoZhanBianMa")%><%=rb.getString("MaoHao")%></label>
			<input id="serialNumber" class="inputDivCss border border-box" minLength="2" maxLength="30" onblur="serialNumberBlur()"/>
			<label id="serialNumber_err" class="inputTipCss errorTipStyle"><%=rb.getString("QingShuRuZhengQueSn")%></label>
		</div>
		<div class="optionDetailsRight">					
			<label class="inputTittleCss"><%=rb.getString("SheBeiZu")%><%=rb.getString("MaoHao")%></label>
			<input id="deviceGroup" maxLength="45" class="inputDivCss border border-box"/>
			<label class="inputTipCss errorTipStyle"></label>
		</div> 	
		<div class='enbaleCommon'>
			<div class="optionDetailsLeft">
				<label class="inputTittleCss"><%=rb.getString("ShengChanChangJia")%><%=rb.getString("MaoHao")%></label>
				<input id="manufacturer" maxLength="45" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div> 
			<div class="optionDetailsRight">
				<label class="inputTittleCss"><%=rb.getString("SheBeiXingHao")%><%=rb.getString("MaoHao")%></label>
				<input id="module_type" maxLength="45" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>
			<div class="optionDetailsLeft">
				<label class="inputTittleCss"><%=rb.getString("HostName")%><%=rb.getString("MaoHao")%></label>
				<input id="host_name" maxLength="50" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>
			<div class="optionDetailsRight">					
				<label class="inputTittleCss"><%=rb.getString("Sheng")%><%=rb.getString("MaoHao")%></label>
				<input id="province" maxLength="64" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div> 
			<div class="optionDetailsLeft">					
				<label class="inputTittleCss"><%=rb.getString("ChengShi")%><%=rb.getString("MaoHao")%></label>
				<input id="city" maxLength="64" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>
			<div class="optionDetailsRight">					
				<label class="inputTittleCss"><%=rb.getString("XianQu")%><%=rb.getString("MaoHao")%></label>
				<input id="district" maxLength="128" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label> 
			</div> 
			<div class="optionDetailsLeft">					
				<label class="inputTittleCss"><%=rb.getString("XiangZhen")%><%=rb.getString("MaoHao")%></label>
				<input id="township" maxLength="128" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>
			<div class="optionDetailsRight">					
				<label class="inputTittleCss"><%=rb.getString("SuoShuWangGe")%><%=rb.getString("MaoHao")%></label>
				<input id="sub_grid" maxLength="64" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>
	 			<div class="optionDetailsLeft">					
				<label class="inputTittleCss"><%=rb.getString("SuoShuZhiJu")%><%=rb.getString("MaoHao")%></label>
				<input id="sub_branches" maxLength="64" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>  
			<div class="optionDetailsRight">					
				<label class="inputTittleCss"><%=rb.getString("ZhanZhiBianMa")%><%=rb.getString("MaoHao")%></label>
				<input id="sub_station_code" maxLength="64" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>					
			<div class="optionDetailsLeft">					
				<label class="inputTittleCss"><%=rb.getString("ZhanZhiMingCheng")%><%=rb.getString("MaoHao")%></label>
				<input id="sub_station_name" maxLength="64" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>					
			<div class="optionDetailsRight">					
				<label class="inputTittleCss"><%=rb.getString("FenBuXiTong")%><%=rb.getString("MaoHao")%></label>
				<input id="sub_distribute_system" maxLength="64" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>					
			<div class="optionDetailsLeft">					
				<label class="inputTittleCss"><%=rb.getString("SuoShuENBBiaoShi")%><%=rb.getString("MaoHao")%></label>
				<input id="sub_cell_id" maxLength="64" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>
	 		<div class="optionDetailsRight">					
				<label class="inputTittleCss"><%=rb.getString("SuoShuENBMingCheng")%><%=rb.getString("MaoHao")%></label>
				<input id="sub_cell_name" maxLength="64" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>  
			<div class="optionDetailsLeft">					
				<label class="inputTittleCss"><%=rb.getString("SuoShuTAList")%><%=rb.getString("MaoHao")%></label>
				<input id="sub_talist" maxLength="64" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>					
			<div class="optionDetailsRight">					
				<label class="inputTittleCss"><%=rb.getString("FuGaiChangJingShuXing")%><%=rb.getString("MaoHao")%></label>
				<input id="over_scen_attr" maxLength="64" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>					
			<div class="optionDetailsLeft">					
				<label class="inputTittleCss"><%=rb.getString("SheBeiGongLv")%><%=rb.getString("MaoHao")%></label>
				<input id="maxtxpower" maxLength="45" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>
	 			<div class="optionDetailsRight">					
				<label class="inputTittleCss"><%=rb.getString("SheBeiJieRuFangShi")%><%=rb.getString("MaoHao")%></label>
				<input id="device_access_mode" maxLength="32" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>  
			<div class="optionDetailsLeft">					
				<label class="inputTittleCss"><%=rb.getString("RXDuanKouShu")%><%=rb.getString("MaoHao")%></label>
				<input id="rx_port_number" maxLength="11" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>					
			<div class="optionDetailsRight">					
				<label class="inputTittleCss"><%=rb.getString("TXDuanKouShu")%><%=rb.getString("MaoHao")%></label>
				<input id="tx_port_number" maxLength="11" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>					
			<div class="optionDetailsLeft">					
				<label class="inputTittleCss"><%=rb.getString("WangGuanIP")%><%=rb.getString("MaoHao")%></label>
				<input id="omc_ip" maxLength="64" onblur='validateIPAddr("omc_ip")' class="inputDivCss border border-box"/>
				<label id="omc_ip_err" class="inputTipCss errorTipStyle"><%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%></label>
			</div>
	 			<div class="optionDetailsRight">					
				<label class="inputTittleCss"><%=rb.getString("SheBeiIP")%><%=rb.getString("MaoHao")%></label>
				<input id="cell_ip" maxLength="64" onblur='validateIPAddr("cell_ip")' class="inputDivCss border border-box"/>
				<label id="cell_ip_err" class="inputTipCss errorTipStyle"><%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%></label>
			</div>  
			<div class="optionDetailsLeft">					
				<label class="inputTittleCss"><%=rb.getString("TAC")%><%=rb.getString("MaoHao")%></label>
				<input id="tac" maxLength="200" onblur='validateTac("tac")' class="inputDivCss border border-box"/>
				<label id="tac_err" class="inputTipCss errorTipStyle">Integer,0-65535</label>
			</div>					
			<div class="optionDetailsRight">					
				<label class="inputTittleCss"><%=rb.getString("RuanJianBanBen")%><%=rb.getString("MaoHao")%></label>
				<input id="software_version" maxLength="500" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>					
			<div class="optionDetailsLeft">					
				<label class="inputTittleCss"><%=rb.getString("WangYuanDengJi")%><%=rb.getString("MaoHao")%></label>
				<input id="net_grade" maxLength="64" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>					
			<div class="optionDetailsRight">					
				<label class="inputTittleCss"><%=rb.getString("AnZhuangJingDu")%><%=rb.getString("MaoHao")%></label>
				<input id="gps_longitude" maxLength="16" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>					
			<div class="optionDetailsLeft">					
				<label class="inputTittleCss"><%=rb.getString("AnZhuangWeiDu")%><%=rb.getString("MaoHao")%></label>
				<input id="gps_latitude" maxLength="16" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>
	 		<div class="optionDetailsRight">					
				<label class="inputTittleCss"><%=rb.getString("AnZhuangXiangXiDiZhi")%><%=rb.getString("MaoHao")%></label>
				<input id="install_address" maxLength="256" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>  
			<div class="optionDetailsLeft">					
				<label class="inputTittleCss"><%=rb.getString("RuWangShiJian")%><%=rb.getString("MaoHao")%></label>
				<input id="access_net_date" class="easyui-datetimebox border border-box" data-options="editable:false"  panelWidth='320px' style="height: 26px;width:320px;"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>					
			<div class="optionDetailsRight">					
				<label class="inputTittleCss"><%=rb.getString("DaiWeiDuiWuHao")%><%=rb.getString("MaoHao")%></label>
				<input id="agent_maintain" maxLength="64" class="inputDivCss border border-box" />
				<label class="inputTipCss errorTipStyle"></label>
			</div>	
			<div class="optionDetailsLeft">					
				<label class="inputTittleCss"><%=rb.getString("YeZhuLianXiRen")%><%=rb.getString("MaoHao")%></label>
				<input id="contact_person" maxLength="64" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>					
			<div class="optionDetailsRight">					
				<label class="inputTittleCss"><%=rb.getString("YeZhuLianXiFangShi")%><%=rb.getString("MaoHao")%></label>
				<input id="contact_number" maxLength="64" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>
			<div class="optionDetailsLeft">					
				<label class="inputTittleCss"><%=rb.getString("SheBeiZhuangTai")%><%=rb.getString("MaoHao")%></label>
				<input id="device_status" maxLength="45" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>					
			<div class="optionDetailsRight">					
				<label class="inputTittleCss"><%=rb.getString("ShangLianKuanDaiZhangHu")%><%=rb.getString("MaoHao")%></label>
				<input id="uplink_broadband_account" maxLength="64" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>
		</div>
		<div class='commonShowOrHide'>
			<div class="optionDetailsLeft">
				<label class="inputTittleCss"><%=rb.getString("HostName")%><%=rb.getString("MaoHao")%></label>
				<input id="host_name" maxLength="50" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>
			<div class="optionDetailsRight">					
				<label id="siteNameLabel" class="inputTittleCss "></label>
				<input id="sub_station_name" maxLength="64" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>	
			<div class="optionDetailsLeft">					
				<label class="inputTittleCss"><%=rb.getString("AnZhuangXiangXiDiZhi")%><%=rb.getString("MaoHao")%></label>
				<input id="install_address" maxLength="256" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div> 
			<div class="optionDetailsRight">					
				<label class="inputTittleCss"><%=rb.getString("YeZhuLianXiFangShi")%><%=rb.getString("MaoHao")%></label>
				<input id="contact_number" maxLength="64" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>
			<div class="optionDetailsLeft">					
				<label id="siteIdLabel" class="inputTittleCss"></label>
				<input id="site_id" maxLength="256" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div> 
			<div class="optionDetailsRight">					
				<label class="inputTittleCss">Circuit Ref.<%=rb.getString("MaoHao")%></label>
				<input id="circuit_ref" maxLength="128" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>
			<div class="optionDetailsLeft">					
				<label class="inputTittleCss">Circuit J & O<%=rb.getString("MaoHao")%></label>
				<input id="circuit_jo" maxLength="128" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div> 
			<div class="optionDetailsRight">					
				<label class="inputTittleCss">Status<%=rb.getString("MaoHao")%></label>
				<input id="service_status" maxLength="50" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div>
			<div class="optionDetailsLeft">					
				<label class="inputTittleCss">Rom<%=rb.getString("MaoHao")%></label>
				<input id="rom" maxLength="50" class="inputDivCss border border-box"/>
				<label class="inputTipCss errorTipStyle"></label>
			</div> 
		</div>
		<input type="submit" style="display:none">
	</div>
   <div>
	   <div class="linkbuttonGroup" style="float:left;margin:0px 55px 20px;">
	        <a href="#" class="linkbutton linkbutton_trend" onclick="addEnb()" id="add"><span><%=rb.getString("QueDing")%></span></a>
	        <a href="#" class="linkbutton linkbutton_nowanna" onclick="CancelAddEnb()"><span><%=rb.getString("QuXiao")%></span></a>
	   </div>
   </div>
</div>

<script>
var groupId = '';
eventBus.$off('addDevice-dialog').$on('addDevice-dialog',function(gId){
	groupId = gId;
});
$(function() {
	//部分字段的显示和隐藏
	if('${enbAdditionalColShow}' == 'true'){
		$('.commonShowOrHide').show();
		$('.enbaleCommon').hide();
		$('#siteIdLabel').html(siteIdLabel);
		$('#siteNameLabel').html(siteNameLabel);
	}else{
		$('.commonShowOrHide').hide();
		$('.enbaleCommon').show();
	}
	$("#deviceGroup").combobox({
			url:'${ctx}/system/deviceGroup/getSimpleDeviceGroupList.action',
	        width: 320,
	        valueField: 'id',
	        textField: 'group_name',
			onLoadSuccess:function(data){
				if(data.length > 0){
					$("#deviceGroup").combobox("setValue",data[0].id);
				}
			}
	 });
	 $("#device_status").combobox({
	        width: 320,
	        valueField: 'value',
	        textField: 'text',
			data:[{text:'<%=rb.getString("RuKu")%>',value:'1'},{text:'<%=rb.getString("AnZhuang")%>',value:'2'},{text:'<%=rb.getString("BaoFei")%>',value:'0'}],
			onLoadSuccess:function(data){
				if(data.length > 0){
					$("#device_status").combobox("setValue",data[0].value);
				}
			}
	 });
});
function serialNumberBlur(){
	var serialNumber = $("#serialNumber").val().trim();
	var temp = /^(\d|[a-zA-Z]|-){1,30}$/;
	if (serialNumber != null && serialNumber.length != 0 && temp.test(serialNumber)) {
		$("#serialNumber_err").css("visibility",'hidden');
	}else{
		$("#serialNumber_err").css("visibility",'visible');
	}	
}
//确认 修改基础指标 KPI 基本信息及门限等
function addEnb() {
	//var selGroup = $("#gridDeviceGroup").datagrid("getSelected");
	var params = {};
	var group_id = $("#deviceGroup").combobox("getValue");
	var device_status = $("#device_status").combobox("getValue");
	params['group_id'] = group_id;
	params['device_status'] = device_status;
	params['timeZone'] = timeZone;
	$("#enbInfo input").each(function(index,ele){
		if(ele.value && ele.id != "access_net_date" && ele.id != "") params[ele.id] = ele.value.trim();
	})
	var access_net_date = $("#access_net_date").combobox('getValue');
	if(access_net_date !=""){
		params['access_net_date'] = access_net_date;
	}
	if(params.serialNumber == undefined || params.serialNumber == ""){
		$("#serialNumber_err").css("visibility",'visible');
		$("#serialNumber").focus();
		return false;
	}
	var isPassFlag = true;
	$('#enbInfo .errorTipStyle').each(function(index,item){
		if($(item).css("visibility") != "hidden"){
			isPassFlag = false;
			return false;
		}
	})
	if(!isPassFlag){
		return;
	}
    //提交请求
   if(!$("#add").hasClass("forbidden")){
		$("#add").addClass("forbidden");
		$.post("${ctx}/system/deviceGroup/addDevice.action", params, function (data) {
	        if (data["success"]) {
	        	showMsg('success_msg',"<%=rb.getString("TianJiaSheBeiChengGong")%>");
				$("#tableDeviceGroupCell").datagrid("reload");
				eventBus.$emit("add-enode")
	        } else {
				$("#add").removeClass("forbidden");
				showMsg('error_msg',data.message); 
	        }
	    }, "json");
	}
	
}
function CancelAddEnb(){
	var changed = false;
	$("#enbInfo input").each(function(index,ele){
		if(ele.value != undefined && ele.value != "") {
			changed = true;
			return false;		
		}
	})
	if(changed){
		$.messager.confirm("<%=rb.getString("TiShi")%>",'<%=rb.getString("QueDingLiKaiDangQianYeMian")%>',function(r){
			if(r){
				$("#enbAddDiv").animate({right:'-1200px'},500,function(){
					$(this).html("");
				})
				eventBus.$emit("add-enode")
			}
		}).addClass("seriousConfirm");		
	}else{
		$("#enbAddDiv").animate({right:'-1200px'},500,function(){
			$(this).html("");
		});
		eventBus.$emit("add-enode")
	}
}
function validateIPAddr(id) {
	var reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/;
	var val = $("#"+id).val();	
	if (val.length > 0 && !reg.test(val)) {
		$("#"+id+"_err").css("visibility",'visible');
	}else{
		$("#"+id+"_err").css("visibility",'hidden');
	}	
}
function validateTac(id){
	var reg = /^(0|[1-9][0-9]*)$/;
	var val = $("#"+id).val();
    if ((val.length > 0 && !reg.test(val)) || (val < 0 || val > 65535)) {
		$("#"+id+"_err").css("visibility",'visible');
	}else{
		$("#"+id+"_err").css("visibility",'hidden');
	}
}
</script>