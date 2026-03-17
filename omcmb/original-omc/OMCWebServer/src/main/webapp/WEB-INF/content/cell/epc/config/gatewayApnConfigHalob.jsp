<%@ page import="java.util.Locale"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page contentType="text/html;charset=UTF-8"%>

<style>
.operationTit,.addIpPoolText{
	position:absolute;
	top:80px;
	right:40px;
	font-size:16px;
	color:#2c8cbf;
	z-index:20000;
	width:32px;
}
.operationTitImport,.importIpPoolText{
	position:absolute;
	top:80px;
	right:-22px;
	font-size:16px;
	color:#2c8cbf;
	z-index:20000;
	width:32px;
}
.epcConfigInfoItemDiv{
	display:inline-block;
	width:400px;
	margin:5px 100px 0 20px;
	vertical-align:top;
	height:90px;
}
.epcConfigInfoItemDiv label{
	display:block;
	line-height:25px;
	color:#797979;
}
.epcConfigInfoItemDiv input{
	width:400px;
}
.epcConfigInfoItemDiv img{
	margin-left:10px;
}
.epcConfigInfoItemDiv .prompt{
	display:block;
	line-height:20px;
	height:20px;
	color: red;
}
.addTrafficSetting ,.addApnSetting,.addIpPoolSetting,.importIpPoolSetting{
	position:absolute;
	display:none;
	top:0px; 
	bottom:0px;
	left:0px;
	right:0px;
	z-index:10;
	background:white;
	padding: 0px 0px 0px 60px;
	overflow-y:auto
}
.chooseArrow{
	display:inline-block;
	margin-left:15px;
}
.epcConfigInfoItem{	
	height:calc(100% - 85px);
	padding-left:60px;
	padding-right:0px;
	margin:20px 0 30px;
}
.importCircle{
	display:inline-block;
	width:22px;
	height:22px;
	background:url("${ctx}/images/import.png") no-repeat;
	margin-left:10px;
	margin-top:10px;
}
.closeCircle{
	display:inline-block;
	width:22px;
	height:22px;
	background:url("${ctx}/images/close.png") no-repeat;
	margin-left:10px;
	margin-top:10px;
}
/* EPC新样式 */
.EPCMainPageCointer{
	position:relative;
	width:100%;
	height:100%; 
}
.importErrTip{
	height:16px;
	width:90%;
	margin-left:20px;
	color:red;
}
.tabsTitle{
	color:#333;
}
</style>
<div class="panelDefault EPCMainPageCointer" id="" style="overflow-x: hidden;">
	<!-- ip config 右上角添加和导入按钮 -->
	<div class="omcTitleButtonGroup" style="top:45px;right:14px;" id="EPCAPNDivButton">
		<div class="omcTitleButtonGroupItem APNaddBtn">
			<span class="titleButtonText"><%=rb.getString("TianJia")%></span>
			<span class="addCircle el-icon el-icon-circle-add" onclick="operEpcConfiguration('ON','addApnSetting')"></span>
		</div>
		<div class="omcTitleButtonGroupItem">
			<span class="titleButtonText"><%=rb.getString("DaoRu")%></span>
			<span class="el-icon el-icon-circle-import" onclick="importEpcConfiguration('ON','importApnSetting')"></span>
		</div>
		<div class="omcTitleButtonGroupItem">
			<span class="titleButtonText"><%=rb.getString("GuanBi")%></span>
			<span class="el-icon el-icon-circle-close" onclick="eventBus.$emit('close-imsi');"></span>
		</div>
	</div>
	<!-- ip池 右上角添加和导入按钮 -->
	<div class="omcTitleButtonGroup" style="top:45px;right:14px;display:none;" id="EPCIPPoolDivButton">
		<div class="omcTitleButtonGroupItem APNIPaddBtn">
			<span class="titleButtonText"><%=rb.getString("TianJia")%></span>
			<span class="addCircle el-icon el-icon-circle-add" onclick="addIpPoolConfig('ON','addIpPoolSetting')"></span>
		</div>
		<div class="omcTitleButtonGroupItem">
			<span class="titleButtonText"><%=rb.getString("DaoRu")%></span>
			<span class="el-icon el-icon-circle-import" onclick="importIpPoolConfig('ON','importIpPoolSetting')"></span>
		</div>
	</div>
	<div class="tabsTitle">
		<span tabtit="EPCAPNDiv" class="active" onclick="swithRightTopButton('EPCAPNDivButton')"><%=rb.getString("APNPeiZhi")%></span>
		<span tabtit="EPCIPPoolDiv" class="" onclick="swithRightTopButton('EPCIPPoolDivButton')"><%=rb.getString("IPChiPeiZhi")%></span>
	</div>
	<div class="tabsContentDiv">	
		<!-- APN 配置 -->
		<div class="EPCAPNDiv" style="display:block;">
			<table id="apn_table"></table>
		</div>
		<!-- IP池 配置 -->
		<div class="EPCIPPoolDiv">
			<table id="apn_table_ipPool"></table>
		</div>
	</div>
	<!-- 添加apn的下拉页 -->
	<div class="addApnSetting" style="z-index:100;overflow-y:auto;padding-top:40px;" id="addApnSetting">	
		<div class="circleIcon" style="top:15px;right:20px;"> 
	    	<span id='eGWRegistAdd' class="el-icon el-icon-circle-close" onclick="operEpcConfiguration('ON','addApnSetting')"></span>
	    	<div class="titleButtonText"><%=rb.getString("GuanBi")%></div>
	    </div>
	    <div>
		  	<div class="epcConfigInfoItemDiv">
		    	<label for="apnName"><%=rb.getString("APNMingCheng")%></label>
		    	<input id="apnName" type="text" maxlength="100" class="easyui-validatebox border border-box"  onblur="if(checkRangLength(this.id,this.value,1,100)){checkName(this.id,this.value);};chengeQCIValue(this.id,this.value);" />
		    	<span id="apnNameCheckSpan" class="prompt" ></span>
		  	</div>
		  	<div class="epcConfigInfoItemDiv">
		    	<label>PDN TYPE</label>
		    	<select id="apnPNDTypeSelect" class="easyui-combobox border border-box" data-options="editable:false" style="height:27px;width:400px;"></select>
		    	<span id="apnPNDTypeSelectCheckSpan" class="prompt" ></span>
		  	</div>
	    </div>
	    <div>
		  	<div class="epcConfigInfoItemDiv">
		    	<label>GW IP ADDRESS</label>
		    	<input id="apnGwIPAddressInput" type="text" maxlength="15" class="easyui-validatebox border border-box" onblur="checkIpv4AddressFormat(this.id,this.value)"/>
		    	<span id="apnGwIPAddressInputCheckSpan" class="prompt" ></span>
		  	</div>
		  	<div class="epcConfigInfoItemDiv">
		    	<label>QCI</label>
		    	<input id="apnQCIInput" type="number" min="5" max="9" onblur="checkMinAndMaxValue(this.id,this.value)" class="easyui-validatebox border border-box"/>
		    	<span id="apnQCIInputCheckSpan" class="prompt" ></span>
		  	</div>
	    </div>
	    <div>
		  	<div class="epcConfigInfoItemDiv">
		    	<label>PRIMARY DNS IPADDR</label>
		    	<input id="apnPrimaryDnsIPInput" type="text" onblur="checkIpv4AddressFormat(this.id,this.value)" class="easyui-validatebox border border-box"/>
		    	<span id="apnPrimaryDnsIPInputCheckSpan" class="prompt" ></span>
		  	</div>
		  	<div class="epcConfigInfoItemDiv">
		    	<label>SECONDARY DNS IPADDR</label>
		    	<input id="apnSecondaryDnsInput" type="text" onblur="checkIpv4AddressFormat(this.id,this.value)" class="easyui-validatebox border border-box" />
		    	<span id="apnSecondaryDnsInputCheckSpan" class="prompt" ></span>
		  	</div>
	    </div>
	    <div>
		  	<div class="epcConfigInfoItemDiv">
		    	<label>ARP PRIORITYLEVEL</label>
		    	<input id="apnPrioritylevelInput" type="number" min="1" max="14" onblur="checkMinAndMaxValue(this.id,this.value)" class="easyui-validatebox border border-box"/>
		    	<span id="apnPrioritylevelInputCheckSpan" class="prompt" ></span>
		  	</div>
		  	<div class="epcConfigInfoItemDiv">
		    	<label>ARP PCI</label>
		    	<select id="apnArpPciInput" class="easyui-combobox border border-box" data-options="editable:false" style="height:27px;width:400px;"></select>
		    	<span id="apnArpPciInputCheckSpan" class="prompt" ></span>
		  	</div>
	    </div>
	    <div>
		  	<div class="epcConfigInfoItemDiv">
		    	<label>ARP PVI</label>
		    	<select id="apnArpPviInput" class="easyui-combobox border border-box" data-options="editable:false" style="height:27px;width:400px;"></select>
		    	<span id="apnArpPviInputCheckSpan" class="prompt" ></span>
		  	</div>
			  <div class="epcConfigInfoItemDiv">
		    	<label for="apnName">T3324(s)</label>
				<input id="tTimer" type="number" min="0" max="255" onblur="checkMinAndMaxValue(this.id,this.value)" class="easyui-validatebox border border-box"/>
		    	<span id="tTimerCheckSpan" class="prompt" ></span>
		  	</div>
	    </div>
		<div>
			<div class="epcConfigInfoItemDiv">
		    	<label for="apnName">eT3412(s)</label>
				<input id="eTimer" type="number" min="0" max="1116000" onblur="checkMinAndMaxValue(this.id,this.value)" class="easyui-validatebox border border-box"/>
		    	<span id="eTimerCheckSpan" class="prompt" ></span>
		  	</div>
			  <div class="epcConfigInfoItemDiv">
				<label>eDRX_PTW</label>
		    	<select id="ptw" class="easyui-combobox border border-box" data-options="editable:false" style="height:27px;width:400px;"></select>
		    	<span id="ptwCheckSpanerror" class="prompt" ></span>
			  </div>
		</div>	
			 <div>
			
			  <div class="epcConfigInfoItemDiv">
				<label>eDRX_VALUE</label>
		    	<select id="edrxvalue" class="easyui-combobox border border-box" data-options="editable:false" style="height:27px;width:400px;"></select>
		    	<span id="edrxvalueCheckSpanerror" class="prompt" ></span>
			  </div>
			  <div class="epcConfigInfoItemDiv">
		    	<label for="apnName">As_IP</label>
				<input id="asip" type="text" onblur="checkIpv4AddressFormat(this.id,this.value)" class="easyui-validatebox border border-box" />
		    	<span id="asipCheckSpan" class="prompt" ></span>
		  	</div>
		</div>	 
	  	<div style="margin-top:15px;margin-bottom:50px;margin-left:20px;">
			<a onclick="addAPNconfigFun()" class="linkbutton"><span><%=rb.getString("BaoCun")%></span></a> 
	  	</div>
	</div>
	
	<!-- 批量导入的下拉页 -->
	<div class="addApnSetting" style="z-index:100;width:96%" id="importApnSetting">	
		<div class="circleIcon" style="top:15px;right:20px;"> 
	    	<span id='eGWRegistAdd' class="el-icon el-icon-circle-close" onclick="importEpcConfiguration('ON','importApnSetting')"></span>
	    	<div class="titleButtonText"><%=rb.getString("GuanBi")%></div>
	    </div>			  
	  	<div style="margin-top:50px;margin-left:20px;">
	    	<input id="filePath" type="text" class="border border-box file_info" readonly="readonly" style="width: 300px;vertical-align:middle;"/>
			<a class='el-icon el-icon-operation-import' style='display:inline-block;margin:0 2px 0 -29px;vertical-align:middle'  title="<%=rb.getString("XuanZeWenJian")%>" href="javascript: void(0)" onclick="scanClick()" style="vertical-align:middle; margin:0 2px 0 -29px;border-left:1px solid #ddd;display:inline-block;width:23px;height:24px;background-color:#fff;"></a>		    
	    	<a onclick="downTemplate()" style="margin-top:-3px;margin-left:20px;" class="linkbutton"><span><%=rb.getString("DaoChuMuBan")%></span></a>
	  	</div>
	    <div id="gwApnNameCheckSpan" class="prompt importErrTip"> </div>
	  	<div class='windowButtonGroup' style="float:left;margin-top:35px;margin-left:20px;">
			<a onclick="importAPNconfigFun()" class="linkbutton linkbutton_trend"><span><%=rb.getString("DaoRu")%></span></a> 
			<a onclick="cancelImportAPNconfigFun('importApnSetting')" class="linkbutton linkbutton_nowanna"><span><%=rb.getString("QuXiao")%></span></a>
	  	</div>
	  	<form enctype="multipart/form-data" method="post" id="uploadForm_apn" style="display: none;"
			action="${ctx}/epc/apnconfig/uploadFile.action?uploadType=APN">
	    	<input name="uploadFile" id="uploadFile" type="file">
	    	<input id="ADDR" name="ADDR" value="">
			<input id="ADDR_PORT" name="ADDR_PORT" value="">
	  	</form>
	  	<form id="downloadFailureApn" style="display:none" method="post" action="${ctx}/epc/apnconfig/downloadFailureFile.action"></form>
	  	<form id="downloadFailureApnIpPool" style="display:none" method="post" action="${ctx}/epc/apnippool/downloadFailureFile.action"></form>
	  	<form id="downloadTemplateApn" style="display:none" method="post"
    				action="${ctx}/epc/apnconfig/downloadImportApnTemplate.action">
	  	</form>
	</div>
	<!-- 添加 ip pool 的下拉页 -->
	<div class="addIpPoolSetting" style="z-index:100;padding-top:40px;" id="addIpPoolSetting">
		<div class="circleIcon" style="top:15px;right:20px;"> 
	    	<span id='eGWRegistAdd' class="el-icon el-icon-circle-close" onclick="addIpPoolConfig('ON','addIpPoolSetting')"></span>
	    	<div class="titleButtonText"><%=rb.getString("GuanBi")%></div>
	    </div>	
	    <div>
		  	<div class="epcConfigInfoItemDiv">
			    <label for="IPPoolapnName"><%=rb.getString("APNMingCheng")%></label>
			    <input id="IPPoolapnName" type="text" maxlength="100" class="easyui-validatebox border border-box"  onblur="if(checkRangLength(this.id,this.value,1,100)){checkName(this.id,this.value);}" />
			    <span id="IPPoolapnNameCheckSpan" class="prompt" ></span>
		  	</div>
		  	<div class="epcConfigInfoItemDiv when1Show">
			    <label>START_SERVED_PARTY_IPV4_ADDRESS</label>
			    <input id="IPPoolstartIpv4AddrInput" type="text" onblur="IPPoolCheckIpv4Format(this.id,this.value)" class="easyui-validatebox border border-box" />
			    <span id="IPPoolstartIpv4AddrInputCheckSpan" class="prompt" ></span>
		  	</div>
	    </div>
	    <div>
		  	<div class="epcConfigInfoItemDiv when1Show">
			    <label>END_SERVED_PARTY_IPV4_ADDRESS</label>
			    <input id="IPPoolEndIpv4AddrInput" type="text" onblur="IPPoolCheckIpv4Format(this.id,this.value)" class="easyui-validatebox border border-box"/>
			    <span id="IPPoolEndIpv4AddrInputCheckSpan" class="prompt" ></span>
		  	</div>
		  	<div class="epcConfigInfoItemDiv when2Show">
			    <label>START_SERVED_PARTY_IPV6_ADDRESS_PREFIX</label>
			    <input id="IPPoolStartIpv6AddrInput" type="text" onblur="IPPoolCheckIpv6Format(this.id,this.value)" class="easyui-validatebox border border-box" />
			    <span id="IPPoolStartIpv6AddrInputCheckSpan" class="prompt" ></span>
		  	</div>
	    </div>
	    <div>
		  	<div class="epcConfigInfoItemDiv when2Show">
			    <label>END_SERVED_PARTY_IPV6_ADDRESS_PREFIX</label>
			    <input id="IPPoolEndIpv6AddrInput" type="text" onblur="IPPoolCheckIpv6Format(this.id,this.value)" class="easyui-validatebox border border-box" />
			    <span id="IPPoolEndIpv6AddrInputCheckSpan" class="prompt" ></span>
		  	</div>
	    </div>			  
	  	<div style="margin-left:20px;margin-top:20px;">
			<a onclick="addIPPoolconfigFun()" class="linkbutton"><span><%=rb.getString("BaoCun")%></span></a> 
	  	</div>
	</div>
	<!-- 批量导入的下拉页 -->
	<div class="importIpPoolSetting" style="z-index:100;width:96%" id="importIpPoolSetting">
		<div class="circleIcon" style="top:15px;right:20px;"> 
	    	<span id='eGWRegistAdd' class="el-icon el-icon-circle-close" onclick="importIpPoolConfig('ON','importIpPoolSetting')"></span>
	    	<div class="titleButtonText"><%=rb.getString("GuanBi")%></div>
	    </div>				  
	  	<div style="margin-top:50px;margin-left:20px;">
		    <input id="filePathIpPool" type="text" class="border border-box file_info" readonly="readonly" style="width: 300px;vertical-align:middle;"/>
			<a class='el-icon el-icon-operation-import' style='display:inline-block;margin:0 2px 0 -29px;vertical-align:middle' title="<%=rb.getString("XuanZeWenJian")%>" href="javascript: void(0)" onclick="scanClickIpPool()" style="vertical-align:middle; margin:0 2px 0 -29px;border-left:1px solid #ddd;display:inline-block;width:23px;height:24px;background-color:#fff;">
		    <a onclick="downTemplateIpPool()" style="margin-top:-3px;margin-left:20px;" class="linkbutton"><span><%=rb.getString("DaoChuMuBan")%></span></a>
	  	</div>
		<div id="ipPoolFileInfo" class="prompt importErrTip"> </div>
	  	<div class='windowButtonGroup' style="float:left;margin-top:35px;margin-left:20px;">
	  		<a onclick="importIPPoolconfigFun()" class="linkbutton linkbutton_trend"><span><%=rb.getString("DaoRu")%></span></a> 
			<a onclick="cancelImportIPPoolconfigFun('importIpPoolSetting')" class="linkbutton linkbutton_nowanna"><span><%=rb.getString("QuXiao")%></span></a>
	  	</div>
	  	<form enctype="multipart/form-data" method="post" id="uploadForm_apn_ip_pool" style="display: none;"
			action="${ctx}/epc/apnippool/uploadFileIpPool.action?uploadType=APN_IP_POOL">
			<input name="uploadFileIP" id="uploadFileIP" type="file">
		    <input id="ADDR_IP" name="ADDR_IP" value="">
   			<input id="ADDR_PORT_IP" name="ADDR_PORT_IP" value="">
	  	</form>
	  	<form id="downloadTemplateIpPool" style="display:none" method="post"
     			action="${ctx}/epc/apnippool/downloadImportApnIpPoolTemplate.action">
	  	</form>
	</div>		
</div>
<!-- 搜索框 -->
<div id="APNsearchDiv" class="omcTableTool" style='padding:10px 0px;'>
	<div class="queryGroup">
		<input id="apn_table_input" placeholder="<%=rb.getString("APNMingCheng")%>"/>
		<b class="el-icon el-icon-common-search" onclick="$('#apn_table').datagrid('reload');"></b>
	</div>
</div>
<!-- 搜索框 -->
<div id="IPsearchDiv" class="omcTableTool" style='padding:10px 0px;'>
	<div class="queryGroup">
		<input id="apn_table_ipPool_input" placeholder="<%=rb.getString("APNMingCheng")%>"/>
		<b class="el-icon el-icon-common-search" onclick="$('#apn_table_ipPool').datagrid('reload');"></b>
	</div>
</div>  
<%-- 窗口-小站详细参数信息 --%>
<div id="winDowloadFailureFile" class="easyui-window" title="T"
     data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:false,width:450,height:180,resizable:false,inline:true">
	<div region="center" data-options="border:false" style="padding:20px">
		<span id="failureText" style="line-height:25px;color:#797979;font-size:14px;"></span>
	</div>
	<div region="south" data-options="border:false" style="height:47px;padding-bottom:20px;">
		<a class="linkbutton" onclick="dowloadFailureFile()" style="float:right;margin-right:20px;"><span><%=rb.getString("XiaZai")%></span></a>
	</div>
</div>	
<!-- 菜单生成 -->
<div id="profile_menu_div"></div>
<script> 
	var ctx = "${ctx}";
	var tianjia="<%=rb.getString("TianJia")%>";
	var guanbi="<%=rb.getString("GuanBi")%>";
	var xuanze="<%=rb.getString("DaoRu")%>";
	// tab 点击
	function swithRightTopButton(ele){
		if(ele == "EPCAPNDivButton"){
			$("#EPCAPNDivButton").show();
			$("#EPCIPPoolDivButton").hide();
		}else{
			$("#EPCAPNDivButton").hide();
			$("#EPCIPPoolDivButton").show();
		}
	}
	/**
	* pdnType 数据格式化
	* @param value{string} 绑定值
	* @param rowData{object}  行数据
	* @param rowIndex{number}  下标
	*/
	function pdnTypeFun(value, rowData, rowIndex){
		if(value == '1'){
	    	return "IPv4";
		}else{
			return "IPv6";
		}
	}
	/**
	* ARP_PCI ARP_PVI 数据格式化
	* @param value{string} 绑定值
	* @param rowData{object}  行数据
	* @param rowIndex{number}  下标
	*/
	function arpSwitchFun(value, rowData, rowIndex){
		if(value == '0'){
	    	return "<%=rb.getString("GuanBi")%>";
		}else{
			return "<%=rb.getString("KaiQi")%>";
		}
	}
	  // 加载完成事件
	var selectedEPCItem = 0;
	$(function(){
		//点击页面其他位置，隐藏操作下拉选项菜单
    $(document).click(function(e){
        var e = e || window.event;
        var elem = e.target || e.srcElement;
        while(elem){
            if($(elem).hasClass('el-icon-operation-more') || elem.className == 'circleBg add_circle' || elem.className == 'toolbarContainer datagrid-toolbar'){
                return
            } 
            elem = elem.parentNode;
        }
        $("#profile_menu_div").css('display','none');
    })
		$("#apn_table_input").keyup(function(event){
			if(event.keyCode==13){
				$('#apn_table').datagrid('reload');
			}
		});
		$("#apn_table_ipPool_input").keyup(function(event){
			if(event.keyCode==13){
				$('#apn_table_ipPool').datagrid('reload');
			}
		});
		$("#apn_table").datagrid({
	        border : false,
	        fit : true,
	        fitColumns: true,
	        url : '${ctx}/epc/apnconfig/getGwApnInfos.action',
	        toolbar:'#APNsearchDiv',
	        singleSelect : true,
	        rownumbers : true,
	        pagination : true,
	        pagePosition : 'bottom',
	        striped : true,
	        columns : [[
						{field:'operation',formatter:profileOpFormatter, width:30,title:''},
						{field:'APN_NAME',width:130,title:'<%=rb.getString("APNMingCheng")%>'},
						{field:'GW_IP_ADDRESS',width:120,title:'GW IP ADDRESS'},
						{field:'QCI',width:100,title:'QCI'},
						{field:'PDN_TYPE',width:100,formatter: pdnTypeFun,title:'PDN TYPE'},
						{field:'PRIMARY_DNS_IPADDR',width:150,title:'PRIMARY DNS IPADDR'},
						{field:'SECONDARY_DNS_IPADDR',width:200,title:'SECONDARY DNS IPADDR'},
						{field:'ARP_PRIORITYLEVEL',width:150,title:'ARP PRIORITYLEVEL'},
						{field:'ARP_PCI',width:125,formatter: arpSwitchFun,title:'ARP PCI'},
						{field:'ARP_PVI',width:125,formatter: arpSwitchFun,title:'ARP PVI'},
		               ]],
		    onBeforeLoad : beforeLoad_apn_table,
	        onLoadSuccess: datagridLoadSuccess,
	        onLoadError : datagridLoadError
	    })
	    $("#apn_table_ipPool").datagrid({
	        border : false,
	        fit : true,
	        fitColumns:true,
	        url : '${ctx}/epc/apnippool/getGwApnIpPoolInfos.action',
	        toolbar:"#IPsearchDiv",
	        singleSelect : true,
	        rownumbers : true,
	        pagination : true,
	        pagePosition : 'bottom',
	        striped : true,
	        columns : [[
						{field:'APN_NAME',width:50,title:'<%=rb.getString("APNMingCheng")%>'},
						{field:'START_SERVED_PARTY_IPV4_ADDRESS',width:100,title:'START SERVED PARTY IPV4 ADDRESS'},
						{field:'END_SERVED_PARTY_IPV4_ADDRESS',width:100,title:'END SERVED PARTY IPV4 ADDRESS'},
						{field:'START_SERVED_PARTY_IPV6_ADDRESS_PREFIX',width:120,title:'START SERVED PARTY IPV6 ADDRESS PREFIX'},
						{field:'END_SERVED_PARTY_IPV6_ADDRESS_PREFIX',width:120,title:'END SERVED PARTY IPV6 ADDRESS PREFIX'},
		            ]],
			onBeforeLoad : beforeLoad_apn_table_ipPool,
	        onLoadSuccess: datagridLoadSuccess,
	        onLoadError : datagridLoadError
	    })
		$("#uploadFile").bind("change", function() {
			$("#filePath").val(this.value);
		});
		$("#uploadFileIP").bind("change", function() {
			$("#filePathIpPool").val(this.value);
		});
		
		$("#importIpPoolFile").bind("change", function() {
			$("#filePathIpPool").val(this.value);
		});
		$("#apnPNDTypeSelect").combobox({
			valueField:'value',
			textField:'text',
			value:"1",
			data:[{
				text:"IPv4",
				value:"1"
			},
			{
				text:"IPv6",
				value:"2"
			}]	
		});
		$("#apnArpPciInput").combobox({
			valueField:'value',
			textField:'text',
			value:"1",
			data:[{
				text:"<%=rb.getString("GuanBi")%>",
				value:"0"
			},
			{
				text:"<%=rb.getString("KaiQi")%>",
				value:"1"
			}]	
		});
		$("#apnArpPviInput").combobox({
			valueField:'value',
			textField:'text',
			value:"1",
			data:[{
				text:"<%=rb.getString("GuanBi")%>",
				value:"0"
			},
			{
				text:"<%=rb.getString("KaiQi")%>",
				value:"1"
			}]	
		});
		$("#ptw").combobox({
			valueField:'value',
			textField:'text',
			value:"1",
			data:[
				{
					text:"WB:1.28S/NB:2.56s",
					value:"1"
				},
				{
					text:"WB:2.56S/NB:5.12s",
					value:"2"
				},
				{
					text:"WB:3.84S/NB:7.68s",
					value:"3"
				},
				{
					text:"WB:5.12S/NB:10.24s",
					value:"4"
				},
				{
					text:"WB:6.4S/NB:12.8s",
					value:"5"
				},
				{
					text:"WB:7.68S/NB:15.36s",
					value:"6"
				},
				{
					text:"WB:8.96S/NB:17.92s",
					value:"7"
				},
				{
					text:"WB:10.24S/NB:20.48s",
					value:"8"
				},
				{
					text:"WB:11.52S/NB:23.04s",
					value:"9"
				},
				{
					text:"WB:12.8S/NB:25.6s",
					value:"10"
				},
				{
					text:"WB:14.08S/NB:28.16s",
					value:"11"
				},
				{
					text:"WB:15.36S/NB:30.72s",
					value:"12"
				},
				{
					text:"WB:16.64S/NB:33.28s",
					value:"13"
				},
				{
					text:"WB:17.92S/NB:35.84s",
					value:"14"
				},
				{
					text:"WB:19.20S/NB:38.4s",
					value:"15"
				},
				{
					text:"WB:20.48S/NB:40.96s",
					value:"16"
				},
			]	
		});
		$("#edrxvalue").combobox({
			valueField:'value',
			textField:'text',
			value:"1",
			data:[
				{
					text:"WB:20.48S/NB:20.48s",
					value:"1"
				},
				{
					text:"WB:40.96S/NB:40.96s",
					value:"2"
				},
				{
					text:"WB:81.92S/NB:81.92s",
					value:"3"
				},
				{
					text:"WB:163.84S/NB:163.84s",
					value:"4"
				},
				{
					text:"WB:327.68S/NB:327.68s",
					value:"5"
				},
				{
					text:"WB:655.36S/NB:655.36s",
					value:"6"
				},
				{
					text:"WB:1310.72S/NB:1310.72s",
					value:"7"
				},
				{
					text:"WB:2621.44S/NB:2621.44s",
					value:"8"
				},
				{
					text:"WB:2621.44S/NB:5242.88s",
					value:"9"
				},
				{
					text:"WB:2621.44S/NB:10485.76s",
					value:"10"
				}
			]	
		});
	});
	/**
* 文件列表 操作 格式化
* @param value{string}   绑定值
* @param row{object}   行数据
* @param index{number}   下标
*/ 
function profileOpFormatter(value,row,index){
	var apnName = row.APN_NAME;
	if(value == null){
		value = "<div class='el-icon el-icon-operation-more'  title='"+ CaoZuo+"' onclick='showProfileMenu(\""+apnName+"\",this)'></div>";
	}
	return value;
}

function showProfileMenu(apnName,el){
	var XiuGai = "<%=rb.getString("XiuGai")%>";
	var data = [
			{apnName: apnName,  code: 'modify', text:XiuGai,cls: 'el-icon el-icon-operation-edit'},
		],
	menuDom = $('#profile_menu_div');
	menuDom.cmenu({data: data, click: profileMenuEvent}); 
	/* 菜单位置 */
	var allHeight = $(document).height(),
		tabsHeight = 0, thisTop = $(el).offset().top;
	if((allHeight - thisTop) <200){
		menuDom.css({
			"top":thisTop - 125 - tabsHeight,
			"left":45,
		});
		if((allHeight - thisTop) <184) $('.item-child ').css({"top":"-54px",});
	}else{
		menuDom.css({
			"top":thisTop - 15 - tabsHeight,
			"left": 45,
		});
	}
	
	menuDom.show();
}
/* 菜单点击事件 */
function profileMenuEvent(row){
	var codes = {/* 映射处理方法  */
			'modify': editInfo ,
		},
		code = row.code;
	
	if(codes[code]) codes[code](row);
	
	$('#profile_menu_div').hide();
}
function editInfo(row){
	var params={
		apnName:row.apnName
	}
	$.post("${ctx}/epc/apnconfig/queryApnInfoByName.action", params, function(data){
		operEpcConfiguration('ON','editApnSetting',data)
	})
	
}
	// APN 请求数据之前触发的事件  赋值操作
  	function beforeLoad_apn_table(param){
	  	var APN_NAME=$("#apn_table_input").val();
	  	if(APN_NAME != ""){
		  	param["APN_NAME"] = APN_NAME;
	  	}
  	}
	// IP 请求数据之前触发的事件  赋值操作
  	function beforeLoad_apn_table_ipPool(param){
	  	var APN_NAME=$("#apn_table_ipPool_input").val();
	  	if(APN_NAME != ""){
		  	param["APN_NAME"] = APN_NAME;
	  	}
  	}
  	// APN 选择导入文件按钮
  	function scanClick() {
	  	$("#gwApnNameCheckSpan").html("");
		$('#uploadFile').click();
 	}
  	// IP 选择导入文件按钮
  	function scanClickIpPool(){
	  	$("#ipPoolFileInfo").html("");
	    $("#uploadFileIP").click();
  	}
	// IP 取消导入
	function cancelImportIPPoolconfigFun(dClass){
		 $("#importIpPoolSetting").slideUp(500);
  		 showImportIppoolFlag = true;
	}
	
	// APN 取消导入
	function cancelImportAPNconfigFun(dclass){
		$("#importApnSetting").slideUp(500);
  		showImportFlag = true;
	}
		
	//APN 下载模板
	function downTemplate(){
		$("#downloadTemplateApn").form("submit",{
			onSubmit: function(param){
				var bool = checkParams(param);
				if(!bool) return false;
			}
		});
	}
	// IP 下载模板
	function downTemplateIpPool(){
		$("#downloadTemplateIpPool").form("submit",{
			onSubmit: function(param){
				var bool = checkParams(param);
				if(!bool) return false;
			}
		});
	}
	
  	//统一添加下拉、关闭事件 根据class判断 tag:on-打开 off-关闭 dclass:操作目标div class名称   
  	var showflag = true;
  	var showImportFlag = true;
  	var showIppoolFlag = true;
  	var showImportIppoolFlag = true;
	var isAdd = true
  	function operEpcConfiguration(tag,dclass,row){
	  	$("#importApnSetting").slideUp(500);
	  	showImportFlag = true;
	  	if(showflag){//添加-打开功能
		  if(dclass === 'addApnSetting'){
			  //初始化界面元素数据
			  $("#addApnSetting").slideDown(500);
			  $(".APNaddBtn .titleButtonText").html("<%=rb.getString("GuanBi")%>");
			  $(".APNaddBtn .addCircle").removeClass("el-icon-circle-add");
			  $(".APNaddBtn .addCircle").addClass("el-icon-circle-close");
			  $("#apnName").val("").attr("disabled",false);
			  $("#apnPNDTypeSelect").combobox("setValue","1");
			  $("#apnGwIPAddressInput").val("");
			  $("#apnQCIInput").val("").attr("disabled",false);
			  $("#apnPrimaryDnsIPInput").val("");
			  $("#apnSecondaryDnsInput").val("");
			  $("#apnPrioritylevelInput").val("");
			  $("#apnArpPciInput").combobox("setValue","1");
			  $("#apnArpPviInput").combobox("setValue","1");
			  $("#addApnSetting .prompt").text("");
			  $("#tTimer").val("");
			  $("#eTimer").val("");
			  $("#asip").val("");
			  $("#edrxvalue").combobox("setValue","");
			  $("#ptw").combobox("setValue","");
			  isAdd = true
		  }else{
			  //初始化界面元素数据
			  $("#addApnSetting").slideDown(500);
			  $(".APNaddBtn .titleButtonText").html("<%=rb.getString("GuanBi")%>");
			  $(".APNaddBtn .addCircle").removeClass("el-icon-circle-add");
			  $(".APNaddBtn .addCircle").addClass("el-icon-circle-close");
			  $("#apnName").val(row.apn_name).attr("disabled",true);
			  $("#apnPNDTypeSelect").combobox("setValue",row.pdn_type);
			  $("#apnGwIPAddressInput").val(row.gw_ip_address);
			  $("#apnQCIInput").val(row.qci).attr("disabled",false);
			  $("#apnPrimaryDnsIPInput").val(row.primary_dns_ipaddr);
			  $("#apnSecondaryDnsInput").val(row.secondary_dns_ipaddr);
			  $("#apnPrioritylevelInput").val(row.arp_prioritylevel);
			  $("#apnArpPciInput").combobox("setValue",row.arp_pci);
			  $("#apnArpPviInput").combobox("setValue",row.arp_pvi);
			  $("#addApnSetting .prompt").text("");
			  $("#tTimer").val(row.t_timer);
			  $("#eTimer").val(row.et_timer);
			  $("#asip").val(row.as_ip);
			  $("#edrxvalue").combobox("setValue",row.edrx_ptw);
			  $("#ptw").combobox("setValue",row.edrx_value);
			  isAdd = false
		  }
			  showflag = false;
	  	}else{
	  		  $("#addApnSetting").slideUp(500);
			  $(".APNaddBtn .titleButtonText").html("<%=rb.getString("TianJia")%>");
			  $(".APNaddBtn .addCircle").addClass("el-icon-circle-add");
			  $(".APNaddBtn .addCircle").removeClass("el-icon-circle-close");
			  showflag = true;
	  	}
  	}
	
  	function importEpcConfiguration(tag,dclass){
	  	$("#gwApnNameCheckSpan").html("");
	  	$("#addApnSetting").slideUp(500);
	  	$(".APNaddBtn .titleButtonText").html("<%=rb.getString("TianJia")%>");
		$(".APNaddBtn .addCircle").addClass("el-icon-circle-add");
		$(".APNaddBtn .addCircle").removeClass("el-icon-circle-close");
		showflag = true;
	  	if(showImportFlag){//添加-打开功能
		  $("#importApnSetting").slideDown(500);
		  $("#filePath").val("");
		  $("#uploadFile").val("");
		  showImportFlag = false;
	  	}else{
	  		$("#importApnSetting").slideUp(500);
	  		showImportFlag = true;
	  	}
  	} 
  	
  	function addIpPoolConfig(tag,dclass){
		if(showIppoolFlag){//添加-打开功能
			  //初始化界面元素数据
			  $("#addIpPoolSetting").slideDown(500);
			  $(".APNIPaddBtn .titleButtonText").html("<%=rb.getString("GuanBi")%>");
			  $(".APNIPaddBtn .addCircle").removeClass("el-icon-circle-add");
			  $(".APNIPaddBtn .addCircle").addClass("el-icon-circle-close");
			  $("#IPPoolapnName").val("");
			  $("#IPPoolstartIpv4AddrInput").val("");
			  $("#IPPoolEndIpv4AddrInput").val("");
			  $("#IPPoolStartIpv6AddrInput").val("");
			  $("#IPPoolEndIpv6AddrInput").val("");
			  $(".when1Show").css("display","inline-block");
			  $("#addIpPoolSetting .prompt").text("");
			  showIppoolFlag = false;
			  $("#importIpPoolSetting").slideUp(500);
		  	  showImportIppoolFlag = true;
		}else{
			  $("#addIpPoolSetting").slideUp(500);
			  $(".APNIPaddBtn .titleButtonText").html("<%=rb.getString("TianJia")%>");
			  $(".APNIPaddBtn .addCircle").addClass("el-icon-circle-add");
			  $(".APNIPaddBtn .addCircle").removeClass("el-icon-circle-close");
			  showIppoolFlag = true;
		}
  	}
  	
  	function importIpPoolConfig(tag,dclass){
	  	$("#ipPoolFileInfo").html("");
	  	if(showImportIppoolFlag){//添加-打开功能
		  //初始化界面元素数据
		  $("#importIpPoolSetting").slideDown(500);
		  $("#filePathIpPool").val("");
		  $("#uploadFileIP").val("");
		  showImportIppoolFlag = false;
		  $("#addIpPoolSetting").slideUp(500);
		  $(".APNIPaddBtn .titleButtonText").html("<%=rb.getString("TianJia")%>");
		  $(".APNIPaddBtn .addCircle").addClass("el-icon-circle-add");
		  $(".APNIPaddBtn .addCircle").removeClass("el-icon-circle-close");
		  showIppoolFlag = true;
	  	}else{
	  		 $("#importIpPoolSetting").slideUp(500);
	  		 showImportIppoolFlag = true;
	  	}
	}
    // APN 新增保存 config过滤配置 
    function addAPNconfigFun(type){
	  	var params={};
		var url = '';
	  	var apnName=$("#apnName").val();
	  	var isError = false;
		if(isAdd){
			url = '${ctx}/epc/apnconfig/addGwApnInfos.action'
		}else{
			url = '${ctx}/epc/apnconfig/updateGwApnInfos.action';
		}
		 
      	if(apnName == ""){
      		if(!checkRangLength("apnName",apnName,1,100)){
	      		isError = true;
	      	}
      	}else{
      		if(!checkName("apnName",apnName)){
          		isError = true;
    		}
      	}
      	params["APN_NAME"] = apnName;
      	params["PDN_TYPE"] = $("#apnPNDTypeSelect").combobox("getValue");
      	var apnGwIPAddressInput = $("#apnGwIPAddressInput").val();
      	if(!checkIpv4AddressFormat("apnGwIPAddressInput",apnGwIPAddressInput)){
      		isError = true;
      	}
     	params["GW_IP_ADDRESS"]=apnGwIPAddressInput;
     	
      	var apnQCIInput = $("#apnQCIInput").val();
      	if(!checkMinAndMaxValue("apnQCIInput",apnQCIInput)){
      		isError = true;
      	}
	  	params["QCI"] = apnQCIInput;
	  	
      	var apnPrimaryDnsIPInput = $("#apnPrimaryDnsIPInput").val();
      	if(!checkIpv4AddressFormat("apnPrimaryDnsIPInput",apnPrimaryDnsIPInput)){
      		isError = true;
      	}
     	params["PRIMARY_DNS_IPADDR"] = apnPrimaryDnsIPInput;
     	
      	var apnSecondaryDnsInput = $("#apnSecondaryDnsInput").val();
      	if(!checkIpv4AddressFormat("apnSecondaryDnsInput",apnSecondaryDnsInput)){
      		isError = true;
      	}
      	params["SECONDARY_DNS_IPADDR"] = apnSecondaryDnsInput;
      	
      	var apnPrioritylevelInput = $("#apnPrioritylevelInput").val();
      	if(!checkMinAndMaxValue("apnPrioritylevelInput",apnPrioritylevelInput)){
      		isError = true;
      	}
	  	params["ARP_PRIORITYLEVEL"] = apnPrioritylevelInput;
     	params["ARP_PCI"] = $("#apnArpPciInput").combobox("getValue");
     	params["ARP_PVI"] = $("#apnArpPviInput").combobox("getValue");
		params["eDRX_PTW"] = $("#ptw").combobox("getValue");
		params["eDRX_VALUE"] = $("#edrxvalue").combobox("getValue");
		var tTimer = $("#tTimer").val();
		if(!checkMinAndMaxValue("tTimer",tTimer)){
      		isError = true;
      	}
		params["T_TIMER"] = tTimer;
		
		var eTimer = $("#eTimer").val();
		if(!checkMinAndMaxValue("eTimer",eTimer)){
      		isError = true;
      	}
		params["eT_TIMER"] = eTimer;

		var asip = $("#asip").val();
      	if(!checkIpv4AddressFormat("asip",asip)){
      		isError = true;
      	}
     	params["As_IP"]=asip;
		var edrxvalue =  $("#edrxvalue").combobox("getValue");
		if (edrxvalue.length === 0) {
			$("#edrxvalueCheckSpanerror").html("<%=rb.getString("QingXuanZe")%>");
			isError = true;
			
		}else{
			$("#edrxvalueCheckSpanerror").html("")
		}

		var ptw =  $("#ptw").combobox("getValue");
		if (ptw.length === 0) {
			$("#ptwCheckSpanerror").html("<%=rb.getString("QingXuanZe")%>");
			isError = true;
			
		}else{
			$("#ptwCheckSpanerror").html('')
		}
		if(isError){return ;}
	
	  	savingCover();
		
		
	  	$.post(url, params, function(data){
		  	cancelSavingCover();
          	if (data["success"]) {
        	  	showMsg('success_msg',"<%=rb.getString("ChengGong")%>");
        	  	$("#apn_table").datagrid("reload");
        	  	$("#addApnSetting").slideUp(500);
        	  	$(".APNaddBtn .titleButtonText").html("<%=rb.getString("TianJia")%>");
   			    $(".APNaddBtn .addCircle").addClass("el-icon-circle-add");
   			    $(".APNaddBtn .addCircle").removeClass("el-icon-circle-close");
  			    showflag = true;
          	} else {
        	  	showMsg('error_msg',data["message"]);
          	}
      	}, "json");
	}
    
	// IP 新增保存
  	function addIPPoolconfigFun(){
		var params = {};
	  	var apnName=$("#IPPoolapnName").val();
	  	var isError = false;
      	if(apnName == ""){
      		if(!checkRangLength("IPPoolapnName",apnName,1,100)){
	      		isError = true;
	      	}
      	}else{
      		if(!checkName("IPPoolapnName",apnName)){
          		isError = true;
    		}
      	}
      	params["APN_NAME"]=apnName;
      	
      	var IPPoolstartIpv4AddrInput = $("#IPPoolstartIpv4AddrInput").val();
      	if(IPPoolstartIpv4AddrInput.length!=0 && !IPPoolCheckIpv4Format(apnName,IPPoolstartIpv4AddrInput)){
    	  	$("#IPPoolstartIpv4AddrInput").text("<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>");
      		isError = true;
      	}
      	var ipv4s = IPPoolstartIpv4AddrInput.replace(".","").replace(".","").replace(".","");
      	params["START_SERVED_PARTY_IPV4_ADDRESS"]=IPPoolstartIpv4AddrInput;
      	
      	var IPPoolEndIpv4AddrInput = $("#IPPoolEndIpv4AddrInput").val();
      	if(IPPoolstartIpv4AddrInput.length!=0 && !IPPoolCheckIpv4Format(apnName,IPPoolEndIpv4AddrInput)){
    	  	$("#IPPoolEndIpv4AddrInput").text("<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>");
      		isError = true;
      	}
      	var ipv4e = IPPoolEndIpv4AddrInput.replace(".","").replace(".","").replace(".","");
      	params["END_SERVED_PARTY_IPV4_ADDRESS"]=IPPoolEndIpv4AddrInput;
      	
      	var IPPoolStartIpv6AddrInput = $("#IPPoolStartIpv6AddrInput").val();
      	if(IPPoolStartIpv6AddrInput.length!=0 && !IPPoolCheckIpv6Format(apnName,IPPoolStartIpv6AddrInput)){
    	  	$("#IPPoolStartIpv6AddrInput").text("<%=rb.getString("QingShuRuHeFaDeIPV6DiZhi")%>");
      		isError = true;
      	}
      	params["START_SERVED_PARTY_IPV6_ADDRESS_PREFIX"]=IPPoolStartIpv6AddrInput;
      	
      	var IPPoolEndIpv6AddrInput = $("#IPPoolEndIpv6AddrInput").val();
      	if(IPPoolEndIpv6AddrInput.length!=0 && !IPPoolCheckIpv6Format(apnName,IPPoolEndIpv6AddrInput)){
    	  	$("#IPPoolEndIpv6AddrInput").text("<%=rb.getString("QingShuRuHeFaDeIPV6DiZhi")%>");
      		isError = true;
      	}
      	params["END_SERVED_PARTY_IPV6_ADDRESS_PREFIX"]=IPPoolEndIpv6AddrInput;
        if(isError){
        	return;
        }
	  	savingCover();
	  	$.post("${ctx}/epc/apnippool/addGwApnIpPoolInfos.action", params, function(data){
		  	cancelSavingCover();
          	if (data["success"]) {
        	  	showMsg('success_msg',"success");
        	  	$("#addIpPoolSetting").slideUp(500);
        	  	$(".APNIPaddBtn .titleButtonText").html("<%=rb.getString("TianJia")%>");
  			    $(".APNIPaddBtn .addCircle").addClass("el-icon-circle-add");
  			    $(".APNIPaddBtn .addCircle").removeClass("el-icon-circle-close");
  			    showIppoolFlag = true;
        	  	$("#apn_table_ipPool").datagrid("reload");
          	} else {
		  		var message = '';
		  		if(typeof data["message"] == 'string') message = JSON.parse(data["message"]);
		  		
          		if(message.APN_NAME){
        		  	$("#IPPoolapnNameCheckSpan").text("<%=rb.getString("APNMingChengBuCunZai")%>");
	        	  	return;
          		}
          		if(message.PDN_TYPE == '1'){
          			if(message.START_IPV4){
        		  	  	$("#IPPoolstartIpv4AddrInputCheckSpan").text("<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>");
          	  		}
          	  		if(message.END_IPV4){
        		  	  	$("#IPPoolEndIpv4AddrInputCheckSpan").text("<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>");
          	  		}
          	  		return;
          		}else{
          			if(message.START_IPV6){
        		  	  	$("#IPPoolStartIpv6AddrInputCheckSpan").text("<%=rb.getString("QingShuRuHeFaDeIPV6DiZhi")%>");
          	  		}
          	  		if(message.END_IPV6){
        		  	  	$("#IPPoolEndIpv6AddrInputCheckSpan").text("<%=rb.getString("QingShuRuHeFaDeIPV6DiZhi")%>");
          	  		}
          	  		return;
          		}
          		if(message.COUNT){
        	  	  	$("#IPPoolapnNameCheckSpan").text("<%=rb.getString("ZuiDaZhiChi32")%>");
          		}
          	}
      	}, "json");
  	}

  	//检查长度
  	function checkRangLength(id,value,minLength,maxLength){
      if(value=="" || value.length<minLength || value.length>maxLength){
		$("#"+id+"CheckSpan").text("<%=rb.getString("QingShuRuDeMingChenZai")%> {"+minLength+"} <%=rb.getString("AND")%> {"+maxLength+"} <%=rb.getString("ChangDu")%>");
		return false;
	  }else{
		$("#"+id+"CheckSpan").text("");
		return true;
	  }
  	}
  
  	//校验IP
  	function isValidIP(ip){
	  var reg =  /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/     
	  return reg.test(ip);     
  	}
  	
	//Ipv6校验  
	function isIPv6(str){ 
		var reg = /^([\da-fA-F]{1,4}:){6}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^::([\da-fA-F]{1,4}:){0,4}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:):([\da-fA-F]{1,4}:){0,3}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){2}:([\da-fA-F]{1,4}:){0,2}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){3}:([\da-fA-F]{1,4}:){0,1}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){4}:((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){7}[\da-fA-F]{1,4}$|^:((:[\da-fA-F]{1,4}){1,6}|:)$|^[\da-fA-F]{1,4}:((:[\da-fA-F]{1,4}){1,5}|:)$|^([\da-fA-F]{1,4}:){2}((:[\da-fA-F]{1,4}){1,4}|:)$|^([\da-fA-F]{1,4}:){3}((:[\da-fA-F]{1,4}){1,3}|:)$|^([\da-fA-F]{1,4}:){4}((:[\da-fA-F]{1,4}){1,2}|:)$|^([\da-fA-F]{1,4}:){5}:([\da-fA-F]{1,4})?$|^([\da-fA-F]{1,4}:){6}:$/
		return reg.test(str);
	}
  	function checkIpv4AddressFormat(id,value){
	  if(!isValidIP(value)){
	     $("#"+id+"CheckSpan").text("<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>");
	     return false;
	  }else{
	     $("#"+id+"CheckSpan").text("");
	     return true;
	  }
  	};
  	function checkIpv6AddressFormat(id,value){
	  if(!isIPv6(value)){
	     $("#"+id+"CheckSpan").text("<%=rb.getString("QingShuRuHeFaDeIPV6DiZhi")%>");
	     return false;
	  }else{
	     $("#"+id+"CheckSpan").text("");
	     return true;
	  }
  	};
  	function IPPoolCheckIpv4Format(id,value){
	  	if(value!==""){
			if(!isValidIP(value)){
				$("#"+id+"CheckSpan").text("<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>");
				return false;
			}else{
				$("#"+id+"CheckSpan").text("");
				return true;
			}
	  	}else{
			$("#"+id+"CheckSpan").text("");
			return true;
  		}
  	};
  	function IPPoolCheckIpv6Format(id,value){
  		if(value!==""){
			if(!isIPv6(value)){
				$("#"+id+"CheckSpan").text("<%=rb.getString("QingShuRuHeFaDeIPV6DiZhi")%>");
				return false;
			}else{
				$("#"+id+"CheckSpan").text("");
				return true;
			}
  		}else{
			$("#"+id+"CheckSpan").text("");
			return true;
  		}
  	};
	//检查整数范围
	function checkIntRange(id,value,minLength,maxLength){
		if(isNumeric(value)&& parseInt(value)>=minLength && parseInt(value)<=maxLength){
			$("#"+id+"CheckSpan").text("");
			return true;
		}else{
			$("#"+id+"CheckSpan").text("<%=rb.getString("QingShuRuYouXiaoDeZhi")%>("+minLength+"~"+maxLength+").");
			return false;
		}
	};
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
  // APN 导入确定
  function importAPNconfigFun(){
	  if($("#filePath").val()==""){
		  $('#uploadFile').click();
		  return;
	  }
	  var filePath = $("#filePath").val();
	  if(!checkUploadFile(filePath)){
		  showMsg('prompt_msg',"<%=rb.getString("WenJianGeShiCuoWu")%>");
		  return;
	  }
	  var param={};
	  savingCover();
	  $("#uploadForm_apn").form("submit", {
			dataType: 'json',
			success: function (data) {
				if(typeof data == 'string') data = eval("(" + data + ")");
				cancelSavingCover();
				if(data["success"] && data["msg"]=="0"){
					$("#failureText").html("<%=rb.getString("BuFenShuJuCuoWuQingChongXingBianJi")%>");
					$("#winDowloadFailureFile").window("setTitle", " <%=rb.getString("XinXi")%>").attr('apnConfig','apn');
					$("#winDowloadFailureFile").window("open");
					$('#apn_table').datagrid('reload');
				}else if(data["success"] && data["msg"]=="1"){
					showMsg('success_msg',"<%=rb.getString("ChengGong")%>");
					cancelImportAPNconfigFun('importApnSetting')
					$('#apn_table').datagrid('reload');
				}else if(data["success"]==false && data["msg"]=="2"){
					if(data["apncount"]=="1024"){
						showMsg('prompt_msg',"<%=rb.getString("APNZuiDaZhiChi1024ChaoYueLeDaoRuXianZhiShuLiang")%>");
					}else{
						showMsg('prompt_msg',"<%=rb.getString("DanChiZuiDaZhiChiDaoRu10")%>");
					}
				}else{
					if (data.msg == "File format is wrong") {
	        			showMsg('prompt_msg',"<%=rb.getString("DaoRuWenJianGeShiTiShi")%>");
					} else {
	        			showMsg('error_msg',"<%=rb.getString("DaoRuShiBai")%>");
	        		}
				}
			},
			onSubmit: function(param){
				var bool = checkParams(param)
				if(!bool) return false;
			}
	  });
  }
  function dowloadFailureFile(){
	  var apnConfig = $("#winDowloadFailureFile").attr('apnConfig');
	  if(apnConfig == 'apn'){
		  $("#downloadFailureApn").form("submit",{
				onSubmit: function(param){
					var bool = checkParams(param);
					if(!bool) return false;
				}
		  });
	  }else{
		  $("#downloadFailureApnIpPool").form("submit",{
				onSubmit: function(param){
					var bool = checkParams(param);
					if(!bool) return false;
				}
		  });
	  }
	  $('#winDowloadFailureFile').window('close');
  }
  // IP 导入确定
  function importIPPoolconfigFun(){
	 var param = {};
	  if($("#filePathIpPool").val()==""){
		  $('#uploadFileIP').click();
		  return;
	  }
	  if($("#filePathIpPoolfilePathIpPool").val()==""){
		  $('#uploadFileIP').click();
		  return;
	  }
	  var filePath = $("#filePathIpPool").val();
	  if(!checkUploadFile(filePath)){
		  showMsg('prompt_msg',"<%=rb.getString("WenJianGeShiCuoWu")%>");
		  return;
	  }
	  var param={};
	  savingCover();
	  $("#uploadForm_apn_ip_pool").form("submit", {
			dataType: 'json',
			success: function (data) {
				if(typeof data == 'string') data = eval("(" + data + ")");
				cancelSavingCover();
				if(data["success"] && data["msg"]=="0"){
					$("#failureText").text("<%=rb.getString("BuFenShuJuCuoWuQingChongXingBianJi")%>");
					$("#winDowloadFailureFile").window("setTitle", " <%=rb.getString("XinXi")%>").attr('apnConfig','IPPool');
					$("#winDowloadFailureFile").window("open");
					$('#apn_table_ipPool').datagrid('reload');
				}else if(data["success"] && data["msg"]=="1"){
					showMsg('success_msg',"<%=rb.getString("CaoZuoChengGong")%>");
					$('#apn_table_ipPool').datagrid('reload');
					cancelImportIPPoolconfigFun('importIpPoolSetting');
				}else if(data["success"]==false && (data["msg"]=="2" || data["msg"]=="3")){
					if(data["msg"]=="2"){
						showMsg('prompt_msg',"<%=rb.getString("BuNengYouDuoGeBuTongDeAPN")%>");
					}else if(data["msg"]=="3"){
						showMsg('prompt_msg',"<%=rb.getString("XiangTongDeAPNZuiDuoZhiChi32GeIPChi")%>");
					}
				}else{
					if (data.msg == "File format is wrong") {
	        			showMsg('prompt_msg',"<%=rb.getString("DaoRuWenJianGeShiTiShi")%>");
	        		} else {
	        			showMsg('error_msg',data.msg);
	        		}
				}
			},
			onSubmit: function(param){
				var bool = checkParams(param);
				if(!bool) return false;
			}
	  });
  }
  	// 文件格式验证
	function checkUploadFile(file){
		var index = file.lastIndexOf(".");
		if(index<0){
			return false;
		}else{
			var ext = file.substring(index+1,file.length);
			if(ext != "xlsx"){
				return false;
			}
		}
		return true;
	}
	//检查名称规则
  	function checkName(id,value){
  		var reg = /^([a-zA-Z0-9]+(\.|-))*[a-zA-Z0-9]+$/
    	if(!reg.test(value)){
			$("#"+id+"CheckSpan").text("<%=rb.getString("ZhiNengShuRuZiMuShuZiXiaHuaXianXiaoShuDian")%>");
			return false;
	  	}else{
			$("#"+id+"CheckSpan").text("");
			return true;
	  	}
  	}
	//检查QCI 和 arp prioritylevel
  	function checkMinAndMaxValue(id,value){
		var minValue = $("#"+id).attr("min")-0;
		var maxValue = $("#"+id).attr("max")-0;
    	if(value=="" || value<minValue || value > maxValue ){
			$("#"+id+"CheckSpan").text( "<%=rb.getString("QingShuRuYiGeShuZi")%>" + minValue+"-"+maxValue);
			return false;
	  	}else{
			$("#"+id+"CheckSpan").text("");
			return true;
	  	}
  	}
	function chengeQCIValue(id,value){
		if(value == "IMS"){
			$("#apnQCIInput").val("5").attr("disabled",true);
		}else{
			$("#apnQCIInput").attr("disabled",false);
		}
	}
</script>