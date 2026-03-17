<%@ page import="java.util.Locale"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<style>
	.tabsTitle{
		border:none;
	}
</style>
<script type="text/javascript">
	var ctx = "${ctx}";
	var tianjia="<%=rb.getString("TianJia")%>";
	var guanbi="<%=rb.getString("GuanBi")%>";
	var xuanze="<%=rb.getString("DaoRu")%>";
</script>

<div class="EPCMainPageCointer">
	<!-- 左侧导航 -->
	<div class="EPCLeftContainer">
		<div class="tabsTitle"><span class='active'><%=rb.getString("APNPeiZhi")%></span></div>
		<!-- 选择APN -->
		<div style="margin:20px 0 0 36px;">
			<select id = "APNchoseEPC" class="easyui-combobox border border-box combobox-f combo-f textbox-f" data-options="editable:false" name="software_version" style="padding-top:0px;height:26px;width:200px;z-index:999"></select>
		</div>
		<!-- PCRF配置列表 -->
		<ul id="PCRFList" class="PCRFList">
			<li class="pitchOnItem"><%=rb.getString("APNPeiZhi")%></li>
			<li><%=rb.getString("IPChiPeiZhi")%></li>
		</ul>
	</div>
	
	<!-- 右侧详细 -->
	<div class="EPCRightContainer">
		<!-- APN 配置 -->
		<div class="EPCRightItem" style="display:block;">
			<div class="tabsTitle"><span class='active'><%=rb.getString("APNPeiZhi")%></span></div>
			<!-- 右上角添加和导入按钮 -->
			<div class="omcTitleButtonGroup" style="top:42px;right:14px;">
				<div class="omcTitleButtonGroupItem APNaddBtn">
					<span class="titleButtonText"><%=rb.getString("TianJia")%></span>
					<span class="el-icon el-icon-circle-add" onclick="operEpcConfiguration('ON','addApnSetting')"></span>
				</div>
				<div class="omcTitleButtonGroupItem">
					<span class="titleButtonText"><%=rb.getString("DaoRu")%></span>
					<span class="el-icon el-icon-circle-import" onclick="importEpcConfiguration('ON','importApnSetting')"></span>
				</div>
			</div>
		
			<!-- 搜索框  以及筛选列 -->
			<div style="position:relative;height:30px;margin-top:26px;">
				<!-- 搜索框 -->
				<div id="APNsearchDiv" class="toolbarContainer">
					<!-- 筛选列按钮 -->
					<a onclick="selectColumns()" style="margin-right:50px;" class="linkbutton"><span><%=rb.getString("ShaiXuanLie")%></span></a>
			  		<div class="queryGroup">
			  			<input id="enbMonitorSearchText" placeholder="<%=rb.getString("APNMingCheng")%>"/>
			  			<b class="el-icon el-icon-common-search" onclick="queryApn()"></b>
			  		</div>
				</div> 
					
				<!-- 筛选列项 -->
				<div class="showHideItem">
					<form>
						<!-- 全选 -->
						<div class="selectAll">
							<input type="checkbox" id="all_contentA_exportConfig" name="all_content1" checked="checked" onclick="allCkOnClickEn(event)"/>
							<label for="all_contentA_exportConfig"><%=rb.getString("QuanXuan")%></label>
						</div>
						<!-- 列表内容 -->
						<ul class="sortul">
							<li class="export1ConfigItem">
								<input type="checkbox" id="_03" item="charging_type" value="charging_type" name="contentA"/>
								<label for="_03">CHARGING TYPE</label>
								<span class="dragHandle"></span>
								<span class="moveTop"></span>
							</li>
							<li class="export1ConfigItem">
								<input type="checkbox" id="_04" item="apn_gx_enable" value="apn_gx_enable" name="contentA"/>
								<label for="_04">APN GX ENABLE</label>
								<span class="dragHandle"></span>
								<span class="moveTop"></span>
							</li>
							<li class="export1ConfigItem">
								<input type="checkbox" id="_05" item="ip_addr_alloc_type" value="ip_addr_alloc_type" name="contentA"/>
								<label for="_05">IP ADDR ALLOC TYPE</label>
								<span class="dragHandle"></span>
								<span class="moveTop"></span>
							</li>
							<li class="export1ConfigItem">
								<input type="checkbox" id="_06" item="extern_server_type" value="extern_server_type" name="contentA"/>
								<label for="_06">EXTERN SERVER TYPE</label>
								<span class="dragHandle"></span>
								<span class="moveTop"></span>
							</li>
							<li class="export1ConfigItem">
								<input type="checkbox" id="_07" item="extern_server_ipaddr" value="extern_server_ipaddr" name="contentA"/>
								<label for="_07">EXTERN SERVER IPADDR</label>
								<span class="dragHandle"></span>
								<span class="moveTop"></span>
							</li>
							<li class="export1ConfigItem">
								<input type="checkbox" id="_08" item="ue_pcef_start_time" value="ue_pcef_start_time" name="contentA"/>
								<label for="_08">UE PCEF START TIME</label>
								<span class="dragHandle"></span>
								<span class="moveTop"></span>
							</li>
							<li class="export1ConfigItem">
								<input type="checkbox" id="_09" item="ue_pcef_end_time" value="ue_pcef_end_time" name="contentA"/>
								<label for="_09">UE PCEF END TIME</label>
								<span class="dragHandle"></span>
								<span class="moveTop"></span>
							</li>
							<li class="export1ConfigItem">
								<input type="checkbox" id="_10" item="primary_dns_ipaddr_type" value="primary_dns_ipaddr_type" name="contentA"/>
								<label for="_10">PRIMARY DNS IPADDR TYPE</label>
								<span class="dragHandle"></span>
								<span class="moveTop"></span>
							</li>
							<li class="export1ConfigItem">
								<input type="checkbox" id="_11" item="primary_dns_ipaddr" value="primary_dns_ipaddr" name="contentA"/>
								<label for="_11">PRIMARY DNS IPADDR</label>
								<span class="dragHandle"></span>
								<span class="moveTop"></span>
							</li>
							<li class="export1ConfigItem">
								<input type="checkbox" id="_12" item="secondary_dns_ipaddr_type" value="secondary_dns_ipaddr_type" name="contentA"/>
								<label for="_12">SECONDARY DNS IPADDR TYPE</label>
								<span class="dragHandle"></span>
								<span class="moveTop"></span>
							</li>
							<li class="export1ConfigItem">
								<input type="checkbox" id="_13" item="secondary_dns_ipaddr" value="secondary_dns_ipaddr" name="contentA"/>
								<label for="_13">SECONDARY DNS IPADDR</label>
								<span class="dragHandle"></span>
								<span class="moveTop"></span>
							</li>
							<li class="export1ConfigItem">
								<input type="checkbox" id="_14" item="offline_charging_type" value="offline_charging_type" name="contentA"/>
								<label for="_14">OFFLINE CHARGING TYPE</label>
								<span class="dragHandle"></span>
								<span class="moveTop"></span>
							</li>
							<li class="export1ConfigItem">
								<input type="checkbox" id="_15" item="cdr_tmp_path" value="cdr_tmp_path" name="contentA"/>
								<label for="_15">OFFLINE CHARGING TYPE</label>
								<span class="dragHandle"></span>
								<span class="moveTop"></span>
							</li>
							<li class="export1ConfigItem">
								<input type="checkbox" id="_16" item="cdr_default_path" value="cdr_default_path" name="contentA"/>
								<label for="_16">CDR DEFAULT PATH</label>
								<span class="dragHandle"></span>
								<span class="moveTop"></span>
							</li>
							<li class="export1ConfigItem">
								<input type="checkbox" id="_17" item="cdr_file_duration" value="cdr_file_duration" name="contentA"/>
								<label for="_17">CDR FILE DURATION</label>
								<span class="dragHandle"></span>
								<span class="moveTop"></span>
							</li>
							<li class="export1ConfigItem">
								<input type="checkbox" id="_18" item="cdr_max_record_limt" value="cdr_max_record_limt" name="contentA"/>
								<label for="_18">CDR MAX RECORD LIMT</label>
								<span class="dragHandle"></span>
								<span class="moveTop"></span>
							</li>
							<li class="export1ConfigItem">
								<input type="checkbox" id="_19" item="cdr_interim_cdr_timer" value="cdr_interim_cdr_timer" name="contentA"/>
								<label for="_19">CDR INTERIM CDR TIMER</label>
								<span class="dragHandle"></span>
								<span class="moveTop"></span>
							</li>
							<li class="export1ConfigItem">
								<input type="checkbox" id="_20" item="max_uplink_data" value="max_uplink_data" name="contentA"/>
								<label for="_20">MAX UPLINK DATA</label>
								<span class="dragHandle"></span>
								<span class="moveTop"></span>
							</li>
							<li class="export1ConfigItem">
								<input type="checkbox" id="_21" item="max_downlink_data" value="max_downlink_data" name="contentA"/>
								<label for="_21">MAX DOWNLINK DATA</label>
								<span class="dragHandle"></span>
								<span class="moveTop"></span>
							</li>
							<li class="export1ConfigItem">
								<input type="checkbox" id="_22" item="primary_cg_server_type" value="primary_cg_server_type" name="contentA"/>
								<label for="_22">PRIMARY CG SERVER TYPE</label>
								<span class="dragHandle"></span>
								<span class="moveTop"></span>
							</li>
							<li class="export1ConfigItem">
								<input type="checkbox" id="_23" item="primary_cg_ipaddr" value="primary_cg_ipaddr" name="contentA"/>
								<label for="_23">PRIMARY CG SERVER</label>
								<span class="dragHandle"></span>
								<span class="moveTop"></span>
							</li>
							<li class="export1ConfigItem">
								<input type="checkbox" id="_24" item="primary_cg_port" value="primary_cg_port" name="contentA"/>
								<label for="_24">PRIMARY CG PORT</label>
								<span class="dragHandle"></span>
								<span class="moveTop"></span>
							</li>
							<li class="export1ConfigItem">
								<input type="checkbox" id="_25" item="secondary_cg_server_type" value="secondary_cg_server_type" name="contentA"/>
								<label for="_25">SECONDARY CG SERVER TYPE</label>
								<span class="dragHandle"></span>
								<span class="moveTop"></span>
							</li>
							<li class="export1ConfigItem">
								<input type="checkbox" id="_26" item="secondary_cg_ipaddr" value="secondary_cg_ipaddr" name="contentA"/>
								<label for="_26">SECONDARY CG IPADDR</label>
								<span class="dragHandle"></span>
								<span class="moveTop"></span>
							</li>
							<li class="export1ConfigItem">
								<input type="checkbox" id="_27" item="secondary_cg_port" value="secondary_cg_port" name="contentA"/>
								<label for="_27">SECONDARY CG PORT</label>
								<span class="dragHandle"></span>
								<span class="moveTop"></span>
							</li>
						</ul>
						<div class="windowButtonGroup" style="float:none !important;margin:20px 0px 20px 81px">			
							<a onclick="ColumnConfigAPN()"   class="linkbutton linkbutton_trend"><span><%=rb.getString("QueDing")%></span></a> 
							<a onclick="closeSelectColumns()"   class="linkbutton linkbutton_nowanna"><span><%=rb.getString("QuXiao")%></span></a> 
						</div>
					</form>
				</div>
			</div>
			<!-- APN配置表格 -->
			<div class="contentDiv">
				<table id="apn_table"></table>
			</div>
			
			<!-- 添加apn的下拉页 -->
			<div class="addApnSetting" style="z-index:51;overflow-y:auto" id="addApnSetting">			  
			  	<div class="shuntChoose" style="margin-top:10px;"></div>
			  	<div class="epcConfigInfoItemDiv">
			    	<label for="gwApnName"><%=rb.getString("APNMingCheng")%></label>
			    	<input id="gwApnName" type="text" value="" maxlength="63" class="easyui-validatebox border border-box"  onblur="if(checkRangLength(this.id,this.value,1,63)){checkName(this.id,this.value);}" />
			    	<span id="gwApnNameCheckSpan" class="prompt" ></span>
			  	</div>
			  	<div class="epcConfigInfoItemDiv">
			    	<label>APN GX ENABLE</label>
			    	<select id="apnGxEnable" name="apnGxEnable" class="easyui-combobox border border-box" data-options="editable:false" style="height:27px;width:400px;">
			    		<option value="2">disabled</option>
			    		<option value="1">enable</option>
			    	</select>
			    	<span id="offlineChargingTypeCheckSpan" class="prompt" ></span>
			  	</div>
			  	<div class="epcConfigInfoItemDiv">
			    	<label>PRIMARY DNS IPADDR</label>
			    	<input id="primaryDnsIpaddr" type="text" maxlength="15" class="easyui-validatebox border border-box" onblur="checkIpAddressFormat(this.id,this.value)"/>
			    	<span id="primaryDnsIpaddrCheckSpan" class="prompt" ></span>
			  	</div>
			  	<div class="epcConfigInfoItemDiv">
			    	<label>SECONDARY DNS IPADDR</label>
			    	<input id="secondaryDnsIpaddr" type="text" maxlength="15" class="easyui-validatebox border border-box" onblur="checkIpAddressFormat(this.id,this.value)"/>
			    	<span id="secondaryDnsIpaddrCheckSpan" class="prompt" ></span>
			  	</div>
			  	<div class="epcConfigInfoItemDiv">
			    	<label>OFFLINE CHARGING TYPE</label>
			    	<select id="offlineChargingType" name="offlineChargingType" class="easyui-combobox border border-box" data-options="editable:false" style="height:27px;width:400px;"></select>
			    	<span id="offlineChargingTypeCheckSpan" class="prompt" ></span>
			  	</div>
			  	<div class="epcConfigInfoItemDiv"">
			    	<label>CDR TMP PATH</label>
			    	<input id="cdrTmpPath" type="text" value="/opt/temp/" maxlength="100" class="easyui-validatebox border border-box" />
			    	<span id="cdrTmpPathCheckSpan" class="prompt" ></span>
			  	</div>
			  	<div class="epcConfigInfoItemDiv">
			    	<label>CDR DEFAULT PATH</label>
			    	<input id="cdrDefaultPath" type="text" value="/opt/temp/" maxlength="100" class="easyui-validatebox border border-box" />
			    	<span id="cdrDefaultPathCheckSpan" class="prompt" ></span>
			  	</div>
			  	<div class="epcConfigInfoItemDiv">
			    	<label>CDR FILE DURATION : <%=rb.getString("FenZhong")%></label>
			    	<input id="cdrFileDuration" type="text" value="60" maxlength="4" onblur="checkCdrFileDuration()" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" />
			    	<span id="cdrFileDurationCheckSpan" class="prompt" ></span>
			  	</div>
			  	<div class="epcConfigInfoItemDiv">
			    	<label>CDR MAX RECORD LIMT : item</label>
			    	<input id="cdrMaxRecordLimt" type="text" value="10" maxlength="2" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" />
			    	<span id="cdrMaxRecordLimtCheckSpan" class="prompt" ></span>
			  	</div>
			  	<div class="epcConfigInfoItemDiv"">
			    	<label>CDR INTERIM CDR TIMER : <%=rb.getString("FenZhong")%></label>
			    	<input id="cdrInterimCdrTimer" type="text" value="60" maxlength="3" onblur="chenkCDrInterTimer()" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" />
			    	<span id="cdrInterimCdrTimerCheckSpan" class="prompt" ></span>
			  	</div>
			  	<div class="epcConfigInfoItemDiv">
			    	<label>MAX UPLINK DATA</label>
			    	<input id="maxUplinkData" type="text" value="0" maxlength="4" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" disabled="disabled"/>
			    	<span id="maxUplinkDataCheckSpan" class="prompt" ></span>
			  	</div>
			  	<div class="epcConfigInfoItemDiv">
			    	<label>MAX DOWNLINK DATA</label>
			    	<input id="maxDownLinkData" type="text" value="0" maxlength="4" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" disabled="disabled"/>
			    	<span id="maxDownLinkDataCheckSpan" class="prompt" ></span>
			  	</div>
			  	<div class="epcConfigInfoItemDiv">
			    	<label>PRIMARY CG SERVER TYPE</label>
			    	<select id="primaryCgServerType" name="primaryCgServerType" class="easyui-combobox border border-box" data-options="editable:false" style="height:27px;width:400px;"></select>
			    	<span id="primaryCgServerTypeCheckSpan" class="prompt" ></span>
			  	</div>
			  	<div class="epcConfigInfoItemDiv">
			    	<label>PRIMARY CG IPADDR</label>
			    	<input id="primaryCgIpaddr" type="text" value=""  class="easyui-validatebox border border-box" />
			    	<span id="primaryCgIpaddrCheckSpan" class="prompt" ></span>
			  	</div>
			  	<div class="epcConfigInfoItemDiv">
			    	<label>SECONDARY CG SERVER TYPE</label>
			    	<select id="secondaryCgServerType" name="secondaryCgServerType" class="easyui-combobox border border-box" data-options="editable:false" style="height:27px;width:400px;"></select>
			    	<span id="secondaryCgServerTypeCheckSpan" class="prompt" ></span>
			  	</div>
			  	<div class="epcConfigInfoItemDiv">
			    	<label>SECONDARY CG IPADDR</label>
			    	<input id="secondaryCgIpaddr" type="text" value=""  class="easyui-validatebox border border-box" />
			    	<span id="secondaryCgIpaddrCheckSpan" class="prompt" ></span>
			  	</div>
			  	<div style="margin-top:15px;margin-bottom:50px;margin-left:20px;">
					<a onclick="addGwApnConfiguration()" class="linkbutton"><span><%=rb.getString("BaoCun")%></span></a> 
			  	</div>
			</div>
			
			<!-- 批量导入的下拉页 -->
			<div class="addApnSetting" style="z-index:51;width:96%" id="importApnSetting">			  
				  	<div class="shuntChoose" style="margin-top:10px;">
					    <ul class="shuntChooseItem">
						</ul>
				  	</div>
				  	<div style="margin-top:40px;margin-left:20px;">
				    	<input id="filePath" type="text" class="border border-box file_info" readonly="readonly" style="width: 300px;vertical-align:middle;"/>
						<a class='titleIcon_import' title="<%=rb.getString("XuanZeWenJian")%>" href="javascript: void(0)" onclick="scanClick()" style="vertical-align:middle; margin:0 2px 0 -29px;border-left:1px solid #ddd;display:inline-block;width:23px;height:24px;background-color:#fff;">
						</a>		    
				    	<span id="gwApnNameCheckSpan" class="prompt" ></span>
				    	<a onclick="downTemplate()" style="margin-top:-3px;margin-left:20px;" class="linkbutton"><span><%=rb.getString("DaoChuMuBan")%></span></a>
				  	</div>
				  	<div class='windowButtonGroup' style="float:left;margin-top:35px;margin-left:20px;">
						<a onclick="impGwApnConfiguration()" class="linkbutton linkbutton_trend"><span><%=rb.getString("DaoRu")%></span></a> 
						<a onclick="cancelGwApnConfiguration('importApnSetting')" class="linkbutton linkbutton_nowanna"><span><%=rb.getString("QuXiao")%></span></a>
				  	</div>
				  	<form enctype="multipart/form-data" method="post" id="uploadForm_apn" style="display: none;"
						action="${ctx}/epc/configuration/uploadFile.action?uploadType=APN">
				    	<input name="uploadFile" id="uploadFile" type="file">
				    	<input id="ADDR" name="ADDR" value="">
	    				<input id="ADDR_PORT" name="ADDR_PORT" value="">
	    				<input id="EPC_ID" name="EPC_ID" value="">
				  	</form>
				  	<form id="downloadFailureApn" style="display:none" method="post" action="${ctx}/epc/configuration/downloadFailureFile.action"></form>
				  	<form id="downloadTemplateApn" style="display:none" method="post"
	      				action="${ctx}/epc/configuration/downloadImportApnTemplate.action">
				  	</form>
				</div>	
		</div>
		
		<!-- IP池配置 -->
		<div class="EPCRightItem">
			<div class="singleTitle"><%=rb.getString("IPChiPeiZhi")%></div>
			<!-- 右上角添加和导入按钮 -->
			<div class="omcTitleButtonGroup" style="top:42px;right:14px;">
				<div class="omcTitleButtonGroupItem APNIPaddBtn">
					<span class="titleButtonText"><%=rb.getString("TianJia")%></span>
					<span class="el-icon el-icon-circle-add" onclick="addIpPoolConfig('ON','addIpPoolSetting')"></span>
				</div>
				<div class="omcTitleButtonGroupItem">
					<span class="titleButtonText"><%=rb.getString("DaoRu")%></span>
					<span class="el-icon el-icon-circle-import" onclick="importIpPoolConfig('ON','importIpPoolSetting')"></span>
				</div>
			</div>
			
			<!-- 搜索框 -->
			<div id="IPsearchDiv" class="toolbarContainer">
		  		<div class="queryGroup">
		  			<input id="enbMonitorSearchTextIpPool" placeholder="<%=rb.getString("APPMingCheng")%>"/>
		  			<b class="el-icon el-icon-common-search" onclick="queryIpPool()"></b>
		  		</div>
			</div> 
					
			<!-- IP池配置表格 -->
			<div class="contentDiv">
				<table id="apn_table_ipPool"></table>
			</div>
			
			<!-- 添加apn的下拉页 -->
			<div class="addIpPoolSetting" style="z-index:51;" id="addIpPoolSetting">			  
			  	<div class="shuntChoose" style="margin-top:10px;"></div>
			  	<div class="epcConfigInfoItemDiv">
				    <label for="gwApnName"><%=rb.getString("APNMingCheng")%></label>
				    <input id="apnName" type="text" value="" maxlength="63" class="easyui-validatebox border border-box"  onblur="if(checkRangLength(this.id,this.value,1,63)){checkName(this.id,this.value);}" />
				    <span id="apnNameCheckSpan" class="prompt" ></span>
			  	</div>
			 	<div class="epcConfigInfoItemDiv">
				    <label>IP ADDR ALLOC TYPE</label>
				    <select id="ipAddrAllocType" name="ipAddrAllocType" class="easyui-combobox border border-box" data-options="editable:false" style="height:27px;width:400px;"></select>
				    <span id="ipAddrAllocTypeCheckSpan" class="prompt" ></span>
			  	</div>
			  	<div class="epcConfigInfoItemDiv when1Show">
				    <label>UE START IPv4 ADDR</label>
				    <input id="ueStartIpv4Addr" type="text" maxlength="16" class="easyui-validatebox border border-box" />
				    <span id="ueStartIpv4AddrCheckSpan" class="prompt" ></span>
			  	</div>
			  	<div class="epcConfigInfoItemDiv when1Show">
				    <label>UE END IPv4 ADDR</label>
				    <input id="ueEndIpv4Addr" type="text" maxlength="16" class="easyui-validatebox border border-box"/>
				    <span id="ueEndIpv4AddrCheckSpan" class="prompt" ></span>
			  	</div>
			  	<div class="epcConfigInfoItemDiv when2Show">
				    <label>UE START IPv6ADDR</label>
				    <input id="ueStartIpv6Addr" type="text" value="" maxlength="128" class="easyui-validatebox border border-box" />
				    <span id="ueStartIpv6AddrCheckSpan" class="prompt" ></span>
			  	</div>
			  	<div class="epcConfigInfoItemDiv when2Show">
				    <label>UE END IPv6ADDR</label>
				    <input id="ueEndIpv6Addr" type="text" value="" maxlength="128" class="easyui-validatebox border border-box" />
				    <span id="ueEndIpv6AddrCheckSpan" class="prompt" ></span>
			  	</div>
			  	<div class="epcConfigInfoItemDiv when2Show">
				    <label>ADDR PREFIX LENGTH</label>
				    <input id="addrPrefixLength" type="text" value="48" maxlength="3" onkeyup="value=value.replace(/[^\d]/g,'')" onblur="checkAddrPreLength()" class="easyui-validatebox border border-box" />
				    <span id="addrPrefixLengthCheckSpan" class="prompt" ></span>
			  	</div>
			  	<div style="margin-left:20px">
					<a onclick="addGwApnIpPoolConfiguration()" class="linkbutton"><span><%=rb.getString("BaoCun")%></span></a> 
			  	</div>
			</div>
			<!-- 批量导入的下拉页 -->
			<div class="importIpPoolSetting" style="z-index:51;width:96%" id="importIpPoolSetting">			  
			  	<div style="margin-top:40px;margin-left:20px;">
				    <input id="filePathIpPool" type="text" class="border border-box file_info" readonly="readonly" style="width: 300px;vertical-align:middle;"/>
				    <span id="ipPoolFileInfo" class="prompt" ></span>
				    <a onclick="downTemplateIpPool()" style="margin-top:-3px;margin-left:20px;" class="linkbutton"><span><%=rb.getString("DaoChuMuBan")%></span></a>
			  	</div>
			  	<div class='windowButtonGroup' style="float:left;margin-top:35px;margin-left:20px;">
			  		<a onclick="impGwApnIpPoolConfiguration()" class="linkbutton linkbutton_trend"><span><%=rb.getString("DaoRu")%></span></a> 
					<a onclick="cancelGwApnIpPoolConfiguration('importIpPoolSetting')" class="linkbutton linkbutton_nowanna"><span><%=rb.getString("QuXiao")%></span></a>
			  	</div>
			  	<form enctype="multipart/form-data" method="post" id="uploadForm_apn_ip_pool" style="display: none;"
					action="${ctx}/epc/configuration/uploadFileIpPool.action?uploadType=APN_IP_POOL">
					<input name="uploadFileIP" id="uploadFileIP" type="file">
				    <input id="ADDR_IP" name="ADDR_IP" value="">
	    			<input id="ADDR_PORT_IP" name="ADDR_PORT_IP" value="">
	    			<input id="EPC_ID_IP" name="EPC_ID_IP" value="">
			  	</form>
			  	<form id="downloadTemplateIpPool" style="display:none" method="post"
	      			action="${ctx}/epc/configuration/downloadImportApnIpPoolTemplate.action">
			  	</form>
			</div>
		</div>
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
<script> 
	<%-- 加载完成事件 --%>
	var columncellAPN = "${cellColumn}";
	var allColumnAPN = "${allColumnAPN}";
	var apn_column = [];
	var selectedEPCItem = 0;
	$(function(){
		var epcServer='${epcServer}';
		var selectEPCData = $.parseJSON(epcServer);
		$("#APNchoseEPC").combobox({
		    	data:selectEPCData,
		    	panelHeight:50,
		        valueField: 'EPC_ID',
		        textField: 'NAME',
		        onSelect:chooseShunt,
		        onLoadSuccess:function(){
		        	$("#APNchoseEPC").combobox("panel").closest(".combo-p").addClass("zIndex");
		        }
		 })
		//新代码
		$("#PCRFList li").click(function(){
			 var _thisindex = $(this).index();
			 $(this).siblings().removeClass("pitchOnItem");
			 $(this).addClass("pitchOnItem");
			 $(".EPCRightItem").eq(_thisindex).css("display","block");
			 $(".EPCRightItem").eq(_thisindex).siblings().css("display","none");
			
			 var selectValue = $("#APNchoseEPC").combobox("getValues");
			 if(selectValue[0] == ""){
				 switch(_thisindex){
				 	case 0:
				 		$("#apn_table").datagrid("resize");
				 		$("#enbMonitorSearchText").val("");
				 		selectedEPCItem = 0;
				 	break;
				 	case 1:
				 		$("#apn_table_ipPool").datagrid("resize");
				 		$("#enbMonitorSearchTextIpPool").val("");
				 		selectedEPCItem = 1;
				 	break;
				 }
			 }else{
				 var paramsArray = $("#APNchoseEPC").combobox("getData");
				 //选中的参数
				 var dataparams = paramsArray.filter(function(item){
				 		return item.EPC_ID == selectValue;
				 }); 
				 var epc_id = dataparams[0].EPC_ID;
				 var epc_ip = dataparams[0].IP;
				 var epc_port = dataparams[0].PORT;
				 var param = {};
				 param["EPC_ID"]=dataparams[0].EPC_ID;
				 param["ADDR"]=dataparams[0].IP;
				 param["ADDR_PORT"]=dataparams[0].PORT;
				 
				 //根据不同的tab加载不同表格
				
				 switch(_thisindex){
				 	case 0:
				 		$("#apn_table").datagrid("resize");
				 		$("#enbMonitorSearchText").val("");
				 		if(!showIppoolFlag){
							  $("#addIpPoolSetting").slideUp(500);
							  $(".APNIPaddBtn .titleButtonText").html("<%=rb.getString("TianJia")%>");
							  $(".APNIPaddBtn .circleBg").addClass("add_circle");
							  $(".APNIPaddBtn .circleBg").removeClass("close_circle");
							  showIppoolFlag = true;
						  };
						  if(!showImportIppoolFlag){
							  $("#importIpPoolSetting").slideUp(500);
						  	  showImportIppoolFlag = true;
						  };
						  //$("#select_epc_apn_val,#select_epc_apn_upload_val").text(name);
						  savingCover();
						  $.post("${ctx}/epc/configuration/addGwApn.action", param, function(data){
							 cancelSavingCover();
							 doSearchUrl('apn_table', param,'${ctx}/epc/configuration/getGwApnInfos.action');
						  }).error(function(){
							 cancelSavingCover();
							 showMsg('error_msg',"load error.");
						  });
				 		selectedEPCItem = 0;
				 	break;
				 	case 1:
				 		$("#apn_table_ipPool").datagrid("resize");
				 		$("#enbMonitorSearchTextIpPool").val("");
				 		 if(!showflag){
							  $("#addApnSetting").slideUp(500);
							  $(".APNaddBtn .titleButtonText").html("<%=rb.getString("TianJia")%>");
							  $(".APNaddBtn .circleBg").addClass("add_circle");
							  $(".APNaddBtn .circleBg").removeClass("close_circle");
							  showflag = true;
						 };
						 
						 if(!showImportFlag){
							 $("#importApnSetting").slideUp(500);
						  	 showImportFlag = true;
						 };
						 
						 $("#select_epc_ip_pool_val,#select_epc_ip_pool_upload_val").text(name);
						 savingCover();
						 $.post("${ctx}/epc/configuration/addGwApnIpPool.action", param, function(data){
							 cancelSavingCover();
							 doSearchUrl('apn_table_ipPool', param,'${ctx}/epc/configuration/getGwApnIpPoolInfos.action');
						 }).error(function(){
							 cancelSavingCover();
							 showMsg('error_msg',"load error.");
						 });
				 		selectedEPCItem = 1;
				 	break;
				 } 
			}
		 })
		
		$("#enbMonitorSearchText").keyup(function(event){
			if(event.keyCode==13){
				queryApn();
			}
		});
		$("#enbMonitorSearchTextIpPool").keyup(function(event){
			if(event.keyCode==13){
				queryIpPool();
			}
		});
		//置顶拖拽功能
		if(allColumnAPN != ""){
			var dataGridItem = $(".export1ConfigItem").clone();
			$(".sortul li").remove();
		    var fakeDataArr = allColumnAPN.split(",");
			 for(var i=0;i<fakeDataArr.length;i++){
				for(var j=0;j<dataGridItem.length;j++){
					if(fakeDataArr[i] == ($(dataGridItem[j]).find("input").attr("item"))){
						$(".sortul").append(dataGridItem[j]);
					}
				}
			}
			 $(".sortul li").first().find(".moveTop").addClass("moveTop_disabled");
		}else{
			$(".sortul li").first().find(".moveTop").addClass("moveTop_disabled");
		};
		var boxes = document.getElementsByName("contentA");
		for(i=0;i<boxes.length;i++){
			boxes[i].checked = true;
		}
		var field_arr = new Array();
		if("" != columncellAPN){
			field_arr = columncellAPN.split(",");
			for(i=0;i<boxes.length;i++){
				for(j=0;j<field_arr.length;j++){
					if(boxes[i].value == field_arr[j]){
						boxes[i].checked = false;
						break;
					}
				}
			}
			document.getElementsByName("all_content1")[0].checked = false;
		}
		ColumnConfigAPN();
		//点击拖拽按钮可拖拽
		$(".sortul").dragsort({
			dragSelectorExclude:".moveTop,input,label", 
			scrollContainer:".sortul",
			dragEnd:savesort
		});
		function savesort(){
			//拖拽为第一个后置顶图标不可用
			//this 不是当前拖拽图标 而是图标的父元素
			if($(this).index() == 0){
				$(this).find(".moveTop").addClass("moveTop_disabled");
				$(this).siblings().find(".moveTop").removeClass("moveTop_disabled");
			}else{
				$(this).find(".moveTop").removeClass("moveTop_disabled");
				$(this).parents(".sortul").find(".export1ConfigItem").eq(0).find(".moveTop").addClass("moveTop_disabled");
			}
		}
		$(".export1ConfigItem").mousedown(function(){
			$(this).addClass("handleMouseDown");
		})
		$(".export1ConfigItem").mouseup(function(){
			$(this).removeClass("handleMouseDown");
		});
		//置顶功能
		$(".moveTop").click(function(){
			var parIndex = $(this).parents("li").index();
			var parentBox = $(this).parents("li");
			if(parIndex != 0){
				$(".sortul").prepend(parentBox);
				$(this).addClass("moveTop_disabled");
				parentBox.siblings().find(".moveTop").removeClass("moveTop_disabled");
			}
		});
		//checkbox的点击事件判断全选是否选中
		$(".sortul input").click(function(){
			if(!$(".sortul input").checked){
				$("#all_contentA_exportConfig").attr("checked",false);
			}
			var allcheckLength = $(".export1ConfigItem").length;
			var checkedboxLength = $("input[type='checkbox'][name = 'contentA']:checked ").length; 
			if(allcheckLength == checkedboxLength){
				$("#all_contentA_exportConfig").prop("checked",true);
			}
		}) 
		
		var pageSizes='${pageSize}';
		pageSizes = Number(pageSizes);
		var pageLists='${pageList}';
		pageLists = pageLists.substring(1,pageLists.length-1);
		pageLists = pageLists.split(",");	
		apn_column = [
						{field:'APN_NAME',title:'<%=rb.getString("APNMingCheng")%>'},
						{field:'CHARGING_TYPE',title:'CHARGING TYPE',formatter: getChargingTypeormatter},
						{field:'APN_GX_ENABLE',title:'APN GX ENABLE'},
						{field:'IP_ADDR_ALLOC_TYPE',title:'IP ADDR ALLOC TYPE',formatter: getALLOCTYPEFormatter},
						{field:'EXTERN_SERVER_TYPE',title:'EXTERN SERVER TYPE',formatter:getIpv4OrIpv6Formatter},
						{field:'EXTERN_SERVER_IPADDR',title:'EXTERN SERVER IPADDR'},
						{field:'UE_PCEF_START_TIME',title:'UE PCEF START TIME'},
						{field:'UE_PCEF_END_TIME',title:'UE PCEF END TIME'},
						{field:'PRIMARY_DNS_IPADDR_TYPE',title:'PRIMARY DNS IPADDR TYPE',formatter:getIpv4OrIpv6Formatter},
						{field:'PRIMARY_DNS_IPADDR',title:'PRIMARY DNS IPADDR'},
						{field:'SECONDARY_DNS_IPADDR_TYPE',title:'SECONDARY DNS IPADDR TYPE',formatter:getIpv4OrIpv6Formatter},
						{field:'SECONDARY_DNS_IPADDR',title:'SECONDARY DNS IPADDR'},
						{field:'OFFLINE_CHARGING_TYPE',title:'OFFLINE CHARGING TYPE',formatter:getOffLineFormatter},
						{field:'CDR_TMP_PATH',title:'CDR TMP PATH'},
						{field:'CDR_DEFAULT_PATH',title:'CDR DEFAULT PATH'},
						{field:'CDR_FILE_DURATION',title:'CDR FILE DURATION'},
						{field:'CDR_MAX_RECORD_LIMIT',title:'CDR MAX RECORD LIMIT'},
						{field:'MAX_UPLINK_DATA',title:'MAX UPLINK DATA'},
						{field:'MAX_DOWNLINK_DATA',title:'MAX DOWNLINK DATA'},
						{field:'PRIMARY_CG_SERVER_TYPE',title:'PRIMARY CG SERVER TYPE',formatter:getIpv4OrIpv6Formatter},
						{field:'PRIMARY_CG_IPADDR',title:'PRIMARY CG IPADDR'},
						{field:'PRIMARY_CG_PORT',title:'PRIMARY CG PORT'},
						{field:'SECONDARY_CG_SERVER_TYPE',title:'SECONDARY CG SERVER TYPE',formatter:getIpv4OrIpv6Formatter},
						{field:'SECONDARY_CG_IPADDR',title:'SECONDARY CG IPADDR'},
						{field:'SECONDARY_CG_PORT',title:'SECONDARY CG PORT'}
		                  ];
		ipPool_column = [
						{field:'NAME',title:'<%=rb.getString("EPCMingChen")%>'},
						{field:'APN_NAME',title:'<%=rb.getString("APNMingCheng")%>'},
						{field:'IP_ADDR_ALLOC_TYPE',title:'IP ADDR ALLOC TYPE',formatter: getAllocFormatter},
						{field:'UE_START_IPV4_ADDR',width:10,title:'UE START IPV4 ADDR'},
						{field:'UE_END_IPV4_ADDR',width:10,title:'UE END IPV4 ADDR'},
						{field:'ADDR_PREFIX_LENGTH',width:10,title:'ADDR PREFIX LENGTH'},
						{field:'UE_START_IPV6ADDR',width:10,title:'UE START IPV6ADDR'},
						{field:'UE_END_IPV6ADDR',width:10,title:'UE END IPV6ADDR'}
		                  ];
		var alllength=0;
	    var hiddenlength=0;
		var show_column = $.extend(true,[],apn_column);
		var show_column_second = $.extend(true,[],ipPool_column);
		for(var index=0;index<show_column.length;index++){
			var column_obj = show_column[index];
			for(var i=0;i<field_arr.length;i++){
				var field = field_arr[i];
				if (field == column_obj.field){
					column_obj.hidden = true;
					if(column_obj.width!=undefined){
						hiddenlength+=column_obj.width;
					}
				}
			}
			if(column_obj.width!=undefined){
				alllength+=column_obj.width;
			}
		}
	    var fitColumn = false;
	    var main_width = $("#gateWayRightDiv").width();
		if((alllength-hiddenlength)<=main_width){
			fitColumn = true;
		}
		
		$("#apn_table").datagrid({
	        border : false,
	        fit : true,
	        fitColumns: fitColumn,
	        url : '${ctx}/epc/configuration/getGwApnInfos.action?EPC_ID= ',
	        singleSelect : true,
	        rownumbers : true,
	        toolbar:'#APNsearchDiv',
	        pageSize : pageSizes,
	        pageList : pageLists,
	        pagination : true,
	        pagePosition : 'bottom',
	        striped : true,
	        columns : [show_column],
	        onLoadError : datagridLoadError
	    })
		
	    $("#apn_table_ipPool").datagrid({
	        border : false,
	        fit : true,
	        fitColumns:true,
	        url : '${ctx}/epc/configuration/getGwApnIpPoolInfos.action?EPC_ID= ',
	        toolbar:"#IPsearchDiv",
	        singleSelect : true,
	        rownumbers : true,
	        pageSize : pageSizes,
	        pageList : pageLists,
	        pagination : true,
	        pagePosition : 'bottom',
	        striped : true,
	        columns : [show_column_second],
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
		$(".shuntChooseItem").empty();
		var epcServer='${epcServer}';
		$.each($.parseJSON(epcServer),function(idx,obj){
			$(".shuntChooseItem").append("<li title='"+obj.IP+"' style='border-bottom: 1px solid #d1ecf5;text-align:left;padding-left:20px;' onclick='chooseShunt(\""+obj.EPC_ID+"\",\""+obj.NAME+"\",\""+obj.IP+"\",\""+obj.PORT+"\",this)'>"+obj.NAME+"</li>");  
		});
		$("#ipAddrAllocType").combobox({
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
			},
			{
				text:"IPv4v6",
				value:"3"
			}],
			onSelect:function(){
				var ipAddrAllocType=$("#ipAddrAllocType").combobox("getValue");
				if(ipAddrAllocType == 1){
					$(".when2Show").css("display","none");
					$(".when1Show").css("display","inline-block");
				}else if(ipAddrAllocType == 2){
					$(".when1Show").css("display","none");
					$(".when2Show").css("display","inline-block");
				}else{
					$(".when1Show").css("display","inline-block");
					$(".when2Show").css("display","inline-block");
				}
			}
		});
		$("#offlineChargingType").combobox({
			valueField:'value',
			textField:'text',
			value:"1",
			data:[{
				text:"time charging",
				value:"1"
			},
			{
				text:"flow charging",
				value:"2"
			},
			{
				text:"both time and flow charging",
				value:"3"
			}],
			onSelect:function(){
				var offlineChargingType=$("#offlineChargingType").combobox("getValue");
				if(offlineChargingType==2||offlineChargingType=="2"){
					$("#cdrInterimCdrTimer").val(0).attr("disabled","disabled");
					$("#cdrInterimCdrTimerCheckSpan").text("");
				}else{
					$("#cdrInterimCdrTimer").val("").removeAttr("disabled");
				}
				if(offlineChargingType==1||offlineChargingType=="1"){
					$("#maxUplinkData,#maxDownLinkData").val(0).attr("disabled","disabled");
					$("#maxUplinkDataCheckSpan,#maxDownLinkDataCheckSpan").text("");
				}else{
					$("#maxUplinkData,#maxDownLinkData").val(100).removeAttr("disabled");
				}
			}
		});
		$("#primaryCgServerType").combobox({
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
		$("#secondaryCgServerType").combobox({
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
	});
	
	//以前的tab切换 需要删除
  	<%-- $("#tabs_service").tabs({
	  onSelect:function(){
		 var tab=$("#tabs_service").tabs('getSelected');
		 var index=$("#tabs_service").tabs('getTabIndex',tab);
		 var addr=$("#APNStting .shuntChooseTit li span").attr("title");
		 var epc_port=$("#APNStting .shuntChooseTit li span").attr("port");
		 var epc_id=$("#APNStting .shuntChooseTit li span").attr("epc_id");
		 var name=$("#APNStting .shuntChooseTit li span").text();
		 var param={};
			 param["EPC_ID"]=epc_id;
			 param["ADDR"]=addr;
			 param["ADDR_PORT"]=epc_port;
		 if(index == 0 ){
			 $("#enbMonitorSearchText").val("");
			  if( epc_id!=undefined && epc_id!="" ){
				  if(!showIppoolFlag){
					  $("#addIpPoolSetting").slideUp(500);
					  $(".addcircleBgIppool .titleButtonText").html("<%=rb.getString("TianJia")%>");
					  $(".addcircleBgIppool .circleBg").addClass("add_circle");
					  $(".addcircleBgIppool .circleBg").removeClass("close_circle");
					  showIppoolFlag = true;
				  };
				  if(!showImportIppoolFlag){
					  $("#importIpPoolSetting").slideUp(500);
				  	  showImportIppoolFlag = true;
				  };
				  
				  $("#select_epc_apn_val,#select_epc_apn_upload_val").text(name);
				  savingCover();
				  $.post("${ctx}/epc/configuration/addGwApn.action", param, function(data){
					 cancelSavingCover();
					 doSearchUrl('apn_table', param,'${ctx}/epc/configuration/getGwApnInfos.action');
				  }).error(function(){
					 cancelSavingCover();
					 $.messager.alert(TiShi, "load error.");
				  });
			  }
		  };
		  if(index == 1 ){
			 $("#enbMonitorSearchTextIpPool").val("");
			 if( epc_id!=undefined && epc_id!="" ){
				 if(!showflag){
					  $("#addApnSetting").slideUp(500);
					  $(".addcircleBg .titleButtonText").html("<%=rb.getString("TianJia")%>");
					  $(".addcircleBg .circleBg").addClass("add_circle");
					  $(".addcircleBg .circleBg").removeClass("close_circle");
					  showflag = true;
				 };
				 
				 if(!showImportFlag){
					 $("#importApnSetting").slideUp(500);
				  	 showImportFlag = true;
				 };
				 
				 $("#select_epc_ip_pool_val,#select_epc_ip_pool_upload_val").text(name);
				 savingCover();
				 $.post("${ctx}/epc/configuration/addGwApnIpPool.action", param, function(data){
					 cancelSavingCover();
					 doSearchUrl('apn_table_ipPool', param,'${ctx}/epc/configuration/getGwApnIpPoolInfos.action');
				 }).error(function(){
					 cancelSavingCover();
					 $.messager.alert(TiShi, "load error.");
				 });
			 }
		  }
	  }
  }) --%>
  var ipAddrAllocType=$("#ipAddrAllocType").combobox("getValue");
  if(ipAddrAllocType == 1){
	  $(".when2Show").css("display","none");
  }else{
	  $(".when1Show").css("display","none");
  }
  
  function getOffLineFormatter(value, rowData, rowIndex){
		if(value=="1"){
			return "time charging";
		}else if(value=="2"){
			return "flow charging";
		}else if(value=="3"){
			return "both time and flow charging";
		}else{
			return value;
		}
	}
  
  	function getChargingTypeormatter(value, rowData, rowIndex){
  		if(value=="0"){
			return "no charging";
		}else if(value=="1"){
			return "offline charging";
		}else if(value=="2"){
			return "online charging";
		}else{
			return value;
		}
  	}
	
	function getALLOCTYPEFormatter(value, rowData, rowIndex){
		if(value=="1"){
			return "static allocate ip";
		}else if(value=="2"){
			return "dynamic allocate ip";
		}else if(value=="3"){
			return "DHCP";
		}else{
			return value;
		}
	}
	
	function getAllocFormatter(value, rowData, rowIndex){
		if(rowData.IP_ADDR_ALLOC_TYPE=="1"){
			return "IPv4";
		}else if(rowData.IP_ADDR_ALLOC_TYPE=="2"){
			return "IPv6";
		}else if(rowData.IP_ADDR_ALLOC_TYPE=="3"){
			return "IPv4&IPv6";
		}else{
			return rowData.IP_ADDR_ALLOC_TYPE;
		}
	}
	
	function getIpv4OrIpv6Formatter(value, rowData, rowIndex){
		if(value=="1"){
			return "IPv4";
		}else if(value=="2"){
			return "IPv6";
		}else{
			return value;
		}
	}
	
  	function operFormatterApn(value, rowData, rowIndex){
		var res="";//epc要求暂时屏蔽 后续他们程序支撑修改删除后 可以放开  pf
		/* 		res="<div class='grid-edit-btn-div' style='margin-left: 15px' title='modify' onclick='modifyApnConfiguration(\""+rowIndex+"\")'></div>";
			res+="<div class='grid-del-btn-div' style='margin-left: 15px' title='delete' onclick='deleteApnConfiguration(\""+rowIndex+"\")'></div>";
		*/
        return res;
  	}
  
  	function queryApn(){
	  	/* var EPC_ID = $("#selectEPCApn").attr("epc_id");
	  	var ADDR = $("#selectEPCApn").attr("title");
	  	var ADDR_PORT=$("#selectEPCApn").attr("port"); */
  		 var selectValue = $("#APNchoseEPC").combobox("getValues");
  	 	 if(selectValue[0] == ""){
  				$('.prompt_msg').html("<%=rb.getString("QingXuanZheEPCFuWu")%>");
  		  		var width = $('.prompt_msg').width();
  		  		$('.prompt_msg').css("left",'50%');
  		  		var left = parseFloat($('.prompt_msg').css("left")) - width/2;
  		  		$('.prompt_msg').css("left",left+'px');
  		  		$('.prompt_msg').animate({top:'55px'},200,function(){
  		  			setTimeout(function(){
  		  				$('.prompt_msg').animate({top:'-40px'},function(){
  		  					$("#APNchoseEPC").combobox("showPanel");
  		  				})
  		  			},3000)
  		  		})
  				return;
  			} 
  		 var paramsArray = $("#APNchoseEPC").combobox("getData");
  		 //选中的参数
  		 var dataparams = paramsArray.filter(function(item){
  		 		return item.EPC_ID == selectValue;
  		 });
  		 var EPC_ID = dataparams[0].EPC_ID;
  		 var ADDR = dataparams[0].IP;
  		 var ADDR_PORT = dataparams[0].PORT;
	  	var APN_NAME=$("#enbMonitorSearchText").val();
	  	<%-- if(EPC_ID == undefined){
		  	$.messager.alert(TiShi, "<%=rb.getString("QingXuanZheEPCFuWu")%>");
		  	$("#APNStting .shuntChooseTit li").first().click();
	  	}else{ --%>
		  	var params={};
		  	params["ADDR"]=ADDR;
		  	params["ADDR_PORT"]=ADDR_PORT;
		  	params["EPC_ID"]=EPC_ID;
		  	var paramReload={};
		  	paramReload["EPC_ID"]=EPC_ID;
		  	paramReload["APN_NAME"]=APN_NAME;
		  	doSearchUrl('apn_table', paramReload,'${ctx}/epc/configuration/getGwApnInfos.action');
	  
  	}
  	
  	function queryIpPool(){
	  	/* var EPC_ID = $("#selectEPCApn").attr("epc_id");
	  	var ADDR = $("#selectEPCApn").attr("title");
	  	var ADDR_PORT=$("#selectEPCApn").attr("port"); */
	  	var selectValue = $("#APNchoseEPC").combobox("getValues");
 	 	 if(selectValue[0] == ""){
 				$('.prompt_msg').html("<%=rb.getString("QingXuanZheEPCFuWu")%>");
 		  		var width = $('.prompt_msg').width();
 		  		$('.prompt_msg').css("left",'50%');
 		  		var left = parseFloat($('.prompt_msg').css("left")) - width/2;
 		  		$('.prompt_msg').css("left",left+'px');
 		  		$('.prompt_msg').animate({top:'55px'},200,function(){
 		  			setTimeout(function(){
 		  				$('.prompt_msg').animate({top:'-40px'},function(){
 		  					$("#APNchoseEPC").combobox("showPanel");
 		  				})
 		  			},3000)
 		  		})
 				return;
 			} 
 		 var paramsArray = $("#APNchoseEPC").combobox("getData");
 		 //选中的参数
 		 var dataparams = paramsArray.filter(function(item){
 		 		return item.EPC_ID == selectValue;
 		 });
 		 var EPC_ID = dataparams[0].EPC_ID;
 		 var ADDR = dataparams[0].IP;
 		 var ADDR_PORT = dataparams[0].PORT;
	  	var APN_NAME=$("#enbMonitorSearchTextIpPool").val();
	  	<%-- if(EPC_ID == undefined){
		  	$.messager.alert(TiShi, "<%=rb.getString("QingXuanZheEPCFuWu")%>");
		  	$("#APNStting .shuntChooseTit li").first().click();
	  	}else{ --%>
		  	var params={};
		  	params["ADDR"]=ADDR;
		  	params["ADDR_PORT"]=ADDR_PORT;
		  	params["EPC_ID"]=EPC_ID;
		  	var paramReload={};
		  	paramReload["EPC_ID"]=EPC_ID;
		  	paramReload["APN_NAME"]=APN_NAME;
		  	doSearchUrl('apn_table_ipPool', paramReload,'${ctx}/epc/configuration/getGwApnIpPoolInfos.action');
	  /* 	}   */
  	}
  	
  	function scanClick() {
		$('#uploadFile').click();
 	}
  	
  	function scanClickIpPool(){
	  $("#uploadFileIP").click();
  	}
  
  	/* 选择列下拉框 */
  	function selectColumns(){
	  $(".showHideItem").slideDown(500);
  	}
  	
  	/* 关闭下拉框 */
  	function closeSelectColumns(){
	  $(".showHideItem").slideUp(500);
  	}
  
  	/* 全选 */
  	function allCkOnClickEn(event) {
		var checked = event.target.checked;
		if (checked) {
			$(".export1ConfigItem input[type='checkbox']").each(function() {
				this.checked = true;
			});
		} else {
			$(".export1ConfigItem input[type='checkbox']").each(function() {
				this.checked = false;
			});
		}
	}
  	
	function cancelGwApnIpPoolConfiguration(dClass){
		/* $("#"+dclass).slideUp(500);
		closeFn("importIpPool","importCircle","importIpPoolText","-22px","importIpPoolConfig",dclass,"ip_close",xuanze); */
		 $("#importIpPoolSetting").slideUp(500);
  		 showImportIppoolFlag = true;
	}
	
	//下载模板
	function cancelGwApnConfiguration(dclass){
		$("#importApnSetting").slideUp(500);
  		showImportFlag = true;
		/* $("#"+dclass).slideUp(500);
		closeFn("addConfigImport","importCircle","operationTitImport","-22px","importEpcConfiguration",dclass,"close",xuanze); */
	}
	
/* 	function cancelGwApnIpPoolConfiguration(dclass){
		$("#"+dclass).slideUp(500);
		closeFn("importIpPool","importCircle","importIpPoolText","-22px","importIpPoolConfig",dclass,"ip_close",xuanze);
	} */
	
	//下载模板
	function downTemplate(){
		var bool = checkForm(document.querySelector('#downloadTemplateApn'));
		if(!bool) return false;
		//$("#downloadTemplateApn").form("submit");
		var url = $("#downloadTemplateApn").attr('action');
		exportByForm(url, {});
	}
	
	function downTemplateIpPool(){
		var bool = checkForm(document.querySelector('#downloadTemplateIpPool'));
		if(!bool) return false;
		//$("#downloadTemplateIpPool").form("submit");
		var url = $("#downloadTemplateIpPool").attr('action');
		exportByForm(url, {});
	}
	
  	//统一添加下拉、关闭事件 根据class判断 tag:on-打开 off-关闭 dclass:操作目标div class名称   
  	var showflag = true;
  	var showImportFlag = true;
  	var showIppoolFlag = true;
  	var showImportIppoolFlag = true;
  	function operEpcConfiguration(tag,dclass){
  		//var epc_id=$("#APNStting .shuntChooseTit li span").attr("epc_id");
  		var epc_id =  $("#APNchoseEPC").combobox('getValue');
		if(epc_id==undefined||epc_id==""){
			$('.prompt_msg').html("<%=rb.getString("QingXuanZheEPCFuWu")%>");
	  		var width = $('.prompt_msg').width();
	  		$('.prompt_msg').css("left",'50%');
	  		var left = parseFloat($('.prompt_msg').css("left")) - width/2;
	  		$('.prompt_msg').css("left",left+'px');
	  		$('.prompt_msg').animate({top:'55px'},200,function(){
	  			setTimeout(function(){
	  				$('.prompt_msg').animate({top:'-40px'},function(){
	  					$("#APNchoseEPC").combobox("showPanel");
	  				})
	  			},3000)
	  		})
  			return;
		} 
	  	//var parentHeight = $("#gateWayRightDiv").height()-75;
	  	//$("#addApnSetting").css('height',parentHeight);
	  	$("#importApnSetting").slideUp(500);
	  	showImportFlag = true;
	  	if(showflag){//添加-打开功能
			/*   if($(".addConfigImport").hasClass("close")){
				  $("#importApnSetting").slideUp(500);
				  closeFn("addConfigImport","importCircle","operationTitImport","-35px","importEpcConfiguration","importApnSetting","close",xuanze);
			  }; */
			  //初始化界面元素数据
			  //clearEPC(dclass);
			  $("#addApnSetting").slideDown(500);
			  $(".APNaddBtn .titleButtonText").html("<%=rb.getString("GuanBi")%>");
			  $(".APNaddBtn .addCircle").removeClass("add_circle");
			  $(".APNaddBtn .addCircle").addClass("close_circle");
			  $("#gwApnName").val("");
			  $("#primaryDnsIpaddr").val("");
			  $("#secondaryDnsIpaddr").val("");
			  $("#offlineChargingType").combobox("setValue","1");
			  $("#cdrTmpPath").val("/opt/temp/");
			  $("#cdrDefaultPath").val("/opt/temp/");
			  $("#cdrFileDuration").val("60");
			  $("#cdrMaxRecordLimt").val("10");
			  $("#cdrInterimCdrTimer").val("60");
			  $("#maxUplinkData").val("0");
			  $("#maxDownLinkData").val("0");
			  $("#primaryCgServerType").combobox("setValue","1");
			  $("#primaryCgIpaddr").val("");
			  $("#secondaryCgServerType").combobox("setValue","1");
			  $("#secondaryCgIpaddr").val("");
			  $("#addApnSetting .prompt").text("");
			  //openFn("addConfig","addCircle","operationTit","40px","operEpcConfiguration",dclass,"close");
			  $(".addApnSetting .shuntChooseItem li").attr("pfs","1");
			  showflag = false;
			 
	  	}else{
	  		  $("#addApnSetting").slideUp(500);
			  $(".APNaddBtn .titleButtonText").html("<%=rb.getString("TianJia")%>");
			  $(".APNaddBtn .addCircle").addClass("add_circle");
			  $(".APNaddBtn .addCircle").removeClass("close_circle");
			  showflag = true;
	  	}
	  	closeSelectColumns();
  	}
	
  	function importEpcConfiguration(tag,dclass){
  		/* var epc_id=$("#APNStting .shuntChooseTit li span").attr("epc_id"); */
		<%-- if(epc_id==undefined||epc_id==""){
			$.messager.alert(TiShi, "<%=rb.getString("QingXuanZheEPCFuWu")%>");
			$("#APNStting .shuntChooseTit li").first().click();
			return;
		} --%>
	 /*  	var parentHeight = $("#gateWayRightDiv").height()-75; */
	  	/* $("#importApnSetting").css('height',parentHeight);
	  	$("#importApnSettingSecond").css('height',parentHeight); */
		var epc_id =  $("#APNchoseEPC").combobox('getValue');
		if(epc_id==undefined||epc_id==""){
			$('.prompt_msg').html("<%=rb.getString("QingXuanZheEPCFuWu")%>");
	  		var width = $('.prompt_msg').width();
	  		$('.prompt_msg').css("left",'50%');
	  		var left = parseFloat($('.prompt_msg').css("left")) - width/2;
	  		$('.prompt_msg').css("left",left+'px');
	  		$('.prompt_msg').animate({top:'55px'},200,function(){
	  			setTimeout(function(){
	  				$('.prompt_msg').animate({top:'-40px'},function(){
	  					$("#APNchoseEPC").combobox("showPanel");
	  				})
	  			},3000)
	  		})
  			return;
		} 
	  	$("#addApnSetting").slideUp(500);
	  	$(".APNaddBtn .titleButtonText").html("<%=rb.getString("TianJia")%>");
		$(".APNaddBtn .addCircle").addClass("add_circle");
		$(".APNaddBtn .addCircle").removeClass("close_circle");
		showflag = true;
	  	if(showImportFlag){//添加-打开功能
		  //初始化界面元素数据
		 /*  if($(".addConfig").hasClass("close")){
			  $("#addApnSetting").slideUp(500);
			  closeFn("addConfig","addCircle","operationTit","40px","operEpcConfiguration","addApnSetting","close",tianjia);
		  }; */
		  $("#importApnSetting").slideDown(500);
		  //clearEPC(dclass);
		  //$("#"+dclass).slideDown(500);
		  $("#filePath").val("");
		  $("#uploadFile").val("");
		 // openFn("addConfigImport","importCircle","operationTitImport","-22px","importEpcConfiguration",dclass,"close");
		  $("#importApnSetting .shuntChooseItem li").attr("pfs","1");
		  showImportFlag = false;
	  	}else{
	  		$("#importApnSetting").slideUp(500);
	  		showImportFlag = true;
	  	}
	 	/* if(tag=="OFF"){
		  $("#"+dclass).slideUp(500);
		  closeFn("addConfigImport","importCircle","operationTitImport","-22px","importEpcConfiguration",dclass,"close",xuanze);
	  	} */
  	} 
  	
  	function addIpPoolConfig(tag,dclass){
  		/* var epc_id=$("#APNStting .shuntChooseTit li span").attr("epc_id"); */
		<%-- if(epc_id==undefined||epc_id==""){
			$.messager.alert(TiShi, "<%=rb.getString("QingXuanZheEPCFuWu")%>");
			$("#APNStting .shuntChooseTit li").first().click();
			return;
		} --%>
		/* var parentHeight = $("#gateWayRightDivIpPool").height()-75; */
		/*  $("#addIpPoolSetting").css('height',parentHeight);  */
		var epc_id =  $("#APNchoseEPC").combobox('getValue');
		if(epc_id==undefined||epc_id==""){
			$('.prompt_msg').html("<%=rb.getString("QingXuanZheEPCFuWu")%>");
	  		var width = $('.prompt_msg').width();
	  		$('.prompt_msg').css("left",'50%');
	  		var left = parseFloat($('.prompt_msg').css("left")) - width/2;
	  		$('.prompt_msg').css("left",left+'px');
	  		$('.prompt_msg').animate({top:'55px'},200,function(){
	  			setTimeout(function(){
	  				$('.prompt_msg').animate({top:'-40px'},function(){
	  					$("#APNchoseEPC").combobox("showPanel");
	  				})
	  			},3000)
	  		})
  			return;
		} 
		if(showIppoolFlag){//添加-打开功能
			  //初始化界面元素数据
			  if($(".importIpPool").hasClass("ip_close")){
				  $("#importIpPoolSetting").slideUp(500);
				  //closeFn("importIpPool","importCircle","importIpPoolText","-22px","importIpPoolConfig","importIpPoolSetting","ip_close",xuanze);
			  };
			  //clearEPC(dclass);
			  $("#addIpPoolSetting").slideDown(500);
			  $(".APNIPaddBtn .titleButtonText").html("<%=rb.getString("GuanBi")%>");
			  $(".APNIPaddBtn .addCircle").removeClass("add_circle");
			  $(".APNIPaddBtn .addCircle").addClass("close_circle");
			  $("#apnName").val("");
			  $("#ueStartIpv4Addr").val("");
			  $("#ueEndIpv4Addr").val("");
			  $("#addrPrefixLength").val("");
			  $("#ueStartIpv6Addr").val("");
			  $("#ueEndIpv6Addr").val("");
			  $("#ipAddrAllocType").combobox("setValue","1");
			  $(".when2Show").css("display","none");
			  $(".when1Show").css("display","inline-block");
			  $("#addIpPoolSetting .prompt").text("");
			  // openFn("addIpPool","addCircle","addIpPoolText","40px","addIpPoolConfig",dclass,"ip_close");
			  $(".addIpPoolSetting .shuntChooseItem li").attr("pfs","1");
			  showIppoolFlag = false;
			  $("#importIpPoolSetting").slideUp(500);
		  	  showImportIppoolFlag = true;
		}else{
			  $("#addIpPoolSetting").slideUp(500);
			  $(".APNIPaddBtn .titleButtonText").html("<%=rb.getString("TianJia")%>");
			  $(".APNIPaddBtn .addCircle").addClass("add_circle");
			  $(".APNIPaddBtn .addCircle").removeClass("close_circle");
			  showIppoolFlag = true;
		}
  	}
  	
  	function importIpPoolConfig(tag,dclass){
  		/* var epc_id=$("#APNStting .shuntChooseTit li span").attr("epc_id"); */
		<%-- if(epc_id==undefined||epc_id==""){
			$.messager.alert(TiShi, "<%=rb.getString("QingXuanZheEPCFuWu")%>");
			$("#APNStting .shuntChooseTit li").first().click();
			return;
		} --%>
	  /* 	var parentHeight = $("#gateWayRightDivIpPool").height()-75; */
	 	/* $("#importIpPoolSetting").css('height',parentHeight); */
		var epc_id =  $("#APNchoseEPC").combobox('getValue');
		if(epc_id==undefined||epc_id==""){
			$('.prompt_msg').html("<%=rb.getString("QingXuanZheEPCFuWu")%>");
	  		var width = $('.prompt_msg').width();
	  		$('.prompt_msg').css("left",'50%');
	  		var left = parseFloat($('.prompt_msg').css("left")) - width/2;
	  		$('.prompt_msg').css("left",left+'px');
	  		$('.prompt_msg').animate({top:'55px'},200,function(){
	  			setTimeout(function(){
	  				$('.prompt_msg').animate({top:'-40px'},function(){
	  					$("#APNchoseEPC").combobox("showPanel");
	  				})
	  			},3000)
	  		})
  			return;
		} 
	  	if(showImportIppoolFlag){//添加-打开功能
		  //初始化界面元素数据
		  if($(".addIpPool").hasClass("ip_close")){
			  $("#addIpPoolSetting").slideUp(500);
			  //closeFn("addIpPool","addCircle","addIpPoolText","40px","addIpPoolConfig","addIpPoolSetting","ip_close",tianjia);
		  };
		  //clearEPC(dclass);
		  $("#importIpPoolSetting").slideDown(500);
		  $("#filePathIpPool").val("");
		  $("#uploadFileIP").val("");
		 // openFn("importIpPool","importCircle","importIpPoolText","-22px","importIpPoolConfig",dclass,"ip_close");
		  $(".importIpPoolSetting .shuntChooseItem li").attr("pfs","1");
		  showImportIppoolFlag = false;
		  $("#addIpPoolSetting").slideUp(500);
		  $(".APNIPaddBtn .titleButtonText").html("<%=rb.getString("TianJia")%>");
		  $(".APNIPaddBtn .addCircle").addClass("add_circle");
		  $(".APNIPaddBtn .addCircle").removeClass("close_circle");
		  showIppoolFlag = true;
	  	}else{
	  		 $("#importIpPoolSetting").slideUp(500);
	  		 showImportIppoolFlag = true;
	  	}
/* 	  	if(tag=="OFF"){
		  $("#"+dclass).slideUp(500);
		  closeFn("importIpPool","importCircle","importIpPoolText","-22px","importIpPoolConfig",dclass,"ip_close",xuanze);
	  	} */
	}
  	
  	function openFn(iconDom,circleDom,textDom,right,fnName,divName,className){
	  $("."+iconDom).children().removeClass(circleDom).addClass("closeCircle");
	  $("."+textDom).text("<%=rb.getString("GuanBi")%>");
	  $("."+textDom).css("right",right);
	  //解除click事件
	  $("."+iconDom).removeAttr("onclick");
	  //重新绑定
	  $("."+iconDom).attr("onclick",""+fnName+"('OFF','"+divName+"')");
	  $("."+iconDom).addClass(className);
  	}
  	
  	function closeFn(iconDom,circleDom,textDom,right,fnName,divName,className,text){
	  $("."+iconDom).children().removeClass("closeCircle").addClass(circleDom);
	  $("."+textDom).text(text);
	  $("."+textDom).css("right",right);
	  //解除click事件
	  $("."+iconDom).removeAttr("onclick");
	  //重新绑定
	  $("."+iconDom).attr("onclick",""+fnName+"('ON','"+divName+"')");
	  $("."+iconDom).removeClass(className);
  	}
  	
  	function clearEPC(dclass){
	  $(".shuntChooseItem").slideUp(250);
	  $(".chooseArrow").attr("src","${ctx}/css/images/bi/eGwArrowDown.png");
	  $(".shuntChooseTit li span").text("<%=rb.getString("XuanZheEPC")%>");
	  /* $("."+dclass+" .epcConfigInfoItemDiv input[class='easyui-validatebox border border-box validatebox-text']").val(""); */
	 /*  $("."+dclass+" .epcConfigInfoItemDiv span[class='prompt']").text(""); */
	  $("."+dclass+" .shuntChooseTit li span").attr("title","");
	  $("."+dclass+" .shuntChooseTit li span").attr("port","");
	  $("."+dclass+" .shuntChooseTit li span").attr("epc_id","");
  	}
  	
	//右上角 添加 /关闭按钮 ，鼠标移入、移出事件 （下方显示提示文字 ）
	$(".addConfig").hover(function(){
		if($(".addConfig").hasClass("close")){
			$(".operationTit").text("<%=rb.getString("GuanBi")%>");
			$(".operationTit").fadeIn(100);
		}else{
			$(".operationTit").text("<%=rb.getString("TianJia")%>");
			$(".operationTit").fadeIn(100);
		}
	},function(){
		$(".operationTit").fadeOut(100);
	})
	$(".addIpPool").hover(function(){
		if($(".addIpPool").hasClass("ip_close")){
			$(".addIpPoolText").text("<%=rb.getString("GuanBi")%>");
			$(".addIpPoolText").fadeIn(100);
		}else{
			$(".addIpPoolText").text("<%=rb.getString("TianJia")%>");
			$(".addIpPoolText").fadeIn(100);
		}
	},function(){
		$(".addIpPoolText").fadeOut(100);
	})
	//右上角导入导出按钮、
	$(".addConfigImport").hover(function(){
		if($(".addConfigImport").hasClass("close")){
			$(".operationTitImport").text("<%=rb.getString("GuanBi")%>");
			$(".operationTitImport").fadeIn(100);
		}else{
			$(".operationTitImport").text("<%=rb.getString("DaoRu")%>");
			$(".operationTitImport").fadeIn(100);
		}
	},function(){
		$(".operationTitImport").fadeOut(100);
	})
	$(".importIpPool").hover(function(){
		if($(".importIpPool").hasClass("ip_close")){
			$(".importIpPoolText").text("<%=rb.getString("GuanBi")%>");
			$(".importIpPoolText").fadeIn(100);
		}else{
			$(".importIpPoolText").text("<%=rb.getString("DaoRu")%>");
			$(".importIpPoolText").fadeIn(100);
		}
	},function(){
		$(".importIpPoolText").fadeOut(100);
	})
	
  	function modifyApnConfiguration(indx){
	  //初始化数据 
	  var row=$("#apn_table").datagrid('getData').rows[indx];
	  $(".modifyApnSetting #configPndName").text(row.NAME+"["+row.EPC_SERVER_IP+"]");
	  $(".modifyApnSetting #configPndName").append('<div class="grid-del-btn-div" style="position:absolute;right:51px;top:43px;" onclick="closeModifyApnSetting()"></div>');
	  
	  $("#m_gwApnName").val(row.APN_NAME);
	  $("#m_ueStartIpv4Addr").val(row.UE_START_IPV4_ADDR);
	  $("#m_ueIpv4NetMask").val(row.UE_IPV4_NET_MASK);
	  $(".modifyApnSetting").animate({right:'1px'},500);
	  	  
  	}
	
  	function deleteApnConfiguration(indx){
	  //初始化数据 
	  var row=$("#apn_table").datagrid('getData').rows[indx];
	  $.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueRenShanChuRenWu")%>", function(r) {
		  if (r) {			  
			  savingCover();
			  $.post("${ctx}/epc/configuration/delApnByID.action", {APN_ID:row.APN_ID,APN_NAME:row.APN_NAME,UE_START_IPV4_ADDR:row.UE_START_IPV4_ADDR,UE_IPV4_NET_MASK:row.UE_IPV4_NET_MASK,ADDR:row.EPC_SERVER_IP,ADDR_PORT:row.EPC_PORT},function(data){
				  cancelSavingCover();
				  if (data["success"]) {
		        	  showMsg('success_msg',"delete success.")
		        	  $("#apn_table").datagrid("reload");
		          } else {
		        	  showMsg('error_msg',data["message"]);
		          }
		      }, "json");
		  }
	    }); 
 	 }
  	
  	function closeModifyApnSetting(){
		var row=$("#apn_table").datagrid('getSelected');
		$("#m_gwApnName").val(row.APN_NAME);
		$("#m_ueStartIpv4Addr").val(row.UE_START_IPV4_ADDR);
		$("#m_ueIpv4NetMask").val(row.UE_IPV4_NET_MASK);
		$(".modifyApnSetting").animate({right:'-1800px'},500);
  	}
  	
  	function openshuntChooseItem(e){
		var isShow = $(".shuntChooseItem").css("display");
		$(".shuntChooseItem").slideToggle(250);
		if(isShow == "none"){
			$(".chooseArrow").attr("src","${ctx}/css/images/bi/eGwArrowUp.png");
		}else{
			$(".chooseArrow").attr("src","${ctx}/css/images/bi/eGwArrowDown.png");			
		}
		e.stopPropagation();
	}
  	//以前添加apn的方法  需要删除
   /*  function chooseShunt(id,name,addr,port,ele){
		$("#enbMonitorSearchText").val("");
		$("#enbMonitorSearchTextIpPool").val("");
		$(".shuntChooseItem").slideUp(250);
		$(".chooseArrow").attr("src","${ctx}/css/images/bi/eGwArrowDown.png");
		var tab=$("#tabs_service").tabs('getSelected');
		var index=$("#tabs_service").tabs('getTabIndex',tab);
		var pfs=$(ele).attr("pfs");
		var param={};
		param["EPC_ID"]=id;
		param["ADDR"]=addr;
		param["ADDR_PORT"]=port;
		$("#APNStting .shuntChooseTit li span").text(name);
		$("#APNStting .shuntChooseTit li span").attr("title",addr);
		$("#APNStting .shuntChooseTit li span").attr("port",port);
		$("#APNStting .shuntChooseTit li span").attr("epc_id",id);
		if(index == 0 && pfs==undefined){
			$("#select_epc_apn_val,#select_epc_apn_upload_val").text(name);
			savingCover();
			 $.post("${ctx}/epc/configuration/addGwApn.action", param, function(data){
				 cancelSavingCover();
				 doSearchUrl('apn_table', param,'${ctx}/epc/configuration/getGwApnInfos.action');
			 }).error(function(){
				 cancelSavingCover();
				 $.messager.alert(TiShi, "load error.");
			 });
		}else if(index ==1 && pfs==undefined){
			$("#select_epc_ip_pool_val,#select_epc_ip_pool_upload_val").text(name);
			savingCover();
			 $.post("${ctx}/epc/configuration/addGwApnIpPool.action", param, function(data){
				 cancelSavingCover();
				 doSearchUrl('apn_table_ipPool', param,'${ctx}/epc/configuration/getGwApnIpPoolInfos.action');
			 }).error(function(){
				 cancelSavingCover();
				 $.messager.alert(TiShi, "load error.");
			 });
		}
	} */
    
    //添加apn config过滤配置 
    function addGwApnConfiguration(){
	  	var params={}
	  	 var selectValue = $("#APNchoseEPC").combobox("getValues");
		 var paramsArray = $("#APNchoseEPC").combobox("getData");
		 //选中的参数
		 var dataparams = paramsArray.filter(function(item){
		 		return item.EPC_ID == selectValue;
		 });
		 var addr=dataparams[0].IP;
		 var epc_port=dataparams[0].PORT; 
		 var epc_id=dataparams[0].EPC_ID; 
	  	/* var addr=$("#APNStting .shuntChooseTit li span").attr("title");
	  	var epc_port=$("#APNStting .shuntChooseTit li span").attr("port");
	  	var epc_id=$("#APNStting .shuntChooseTit li span").attr("epc_id"); */
	  	params["EPC_SERVER_IP"]=addr;
	  	params["EPC_PORT"]=epc_port;
	  	params["EPC_ID"]=epc_id;
	  
	  	var apnName=$("#gwApnName").val();
      	if(!checkRangLength("gwApnName",apnName,1,63)){
    	  	return;
      	}
      	if(!checkName("gwApnName",apnName)){
			return;
		}
      	/*
      	var reg = /^([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9_.-][a-zA-Z0-9]+)$/
      	$("#gwApnNameCheckSpan").text("");
  	  	if(!reg.test(apnName)){
  			$("#gwApnNameCheckSpan").text("<%=rb.getString("ZhiNengShuRuZiMuShuZiXiaHuaXianXiaoShuDian")%>");
  		  	return;
  	  	}
  	  	*/
  	  	if(checkApnNameConfig(epc_id,apnName)>0){
  			$("#gwApnNameCheckSpan").text("<%=rb.getString("APNMingChengYiCunZai")%>");
		  	return;
  	  	}
  	  	if(getApnConfigCount(epc_id)>=1024){
  			$("#gwApnNameCheckSpan").text("<%=rb.getString("APNZuiDaZhiNengPeiZhi1024Ge")%>");
		 	return;
  	  	}
      	params["APN_NAME"]=apnName;
      	var primaryDnsIpaddr = $("#primaryDnsIpaddr").val();
      	if(!checkIpAddressFormat("primaryDnsIpaddr",primaryDnsIpaddr)){
    	  	return;
      	}
     	params["PRIMARY_DNS_IPADDR"]=primaryDnsIpaddr;
      	var secondaryDnsIpaddr = $("#secondaryDnsIpaddr").val();
      	if(!checkIpAddressFormat("secondaryDnsIpaddr",secondaryDnsIpaddr)){
    	  	return;
      	}
      	params["SECONDARY_DNS_IPADDR"]=secondaryDnsIpaddr;
      	var cdrTmpPath = $("#cdrTmpPath").val();
      	var reg = /^(\/[\w-]+)*\/$/;
      	$("#cdrTmpPathCheckSpan").text("");
      	if(cdrTmpPath.length!=0 && !reg.test(cdrTmpPath)){
    	  	$("#cdrTmpPathCheckSpan").text("<%=rb.getString("WenJianLuJinGeShiCuoWu")%>");
    	  	return;
      	}
      	params["CDR_TMP_PATH"]=cdrTmpPath;
      
      	var cdrDefaultPath = $("#cdrDefaultPath").val();
      	var reg = /^(\/[\w_]+)*\/$/;
      	$("#cdrDefaultPathCheckSpan").text("");
      	if(cdrDefaultPath.length!=0 && !reg.test(cdrDefaultPath)){
    	  	$("#cdrDefaultPathCheckSpan").text("");
    	  	return;
      	}
      	params["CDR_DEFAULT_PATH"]=cdrDefaultPath;
      
      	var cdrMaxRecordLimt = $("#cdrMaxRecordLimt").val();
      	if(!checkIntRange("cdrMaxRecordLimt",cdrMaxRecordLimt,1,50)){
    	  	return;
      	}
      	params["CDR_MAX_RECORD_LIMT"]=cdrMaxRecordLimt;
      
      	var cdrFileDuration = $("#cdrFileDuration").val();
      	if(!checkIntRange("cdrFileDuration",cdrFileDuration,60,1440)){
    	  	return;
      	}
      	$("#cdrFileDurationCheckSpan").text("");
      	if(!(parseInt(cdrFileDuration,10)%60==0)){
    	  	$("#cdrFileDurationCheckSpan").text("<%=rb.getString("ZhiNengShi60DeBeiShuZuiDa60ChengYi24")%>");//Must be a multiple of 60,Max 60*24
    	  	return;
      	}
      	params["CDR_FILE_DURATION"]=cdrFileDuration;
      
      	var offlineChargingType=$("#offlineChargingType").combobox("getValue");
      	var cdrInterimCdrTimer = $("#cdrInterimCdrTimer").val();
	  	if($.trim(cdrInterimCdrTimer).length!=0&&offlineChargingType!=2&&offlineChargingType!="2"&&!checkIntRange("cdrInterimCdrTimer",cdrInterimCdrTimer,1,120)){
		 	return;
	  	}
	  	params["OFFLINE_CHARGING_TYPE"]=offlineChargingType;
	  	params["CDR_INTERIM_CDR_TIMER"]=cdrInterimCdrTimer;
	  	var maxUplinkData = $("#maxUplinkData").val();
	  	if($.trim(maxUplinkData).length!=0&&offlineChargingType!=1&&offlineChargingType!="1"&&!checkIntRange("maxUplinkData",maxUplinkData,1,1024)){
		 	return;
	  	}
	  	params["MAX_UPLINK_DATA"]=maxUplinkData;
	  
	  	var maxDownLinkData = $("#maxDownLinkData").val();
	  	if($.trim(maxDownLinkData).length!=0&&offlineChargingType!=1&&offlineChargingType!="1"&&!checkIntRange("maxDownLinkData",maxDownLinkData,1,1024)){
		 	return;
	  	}
	  	params["MAX_DOWNLINK_DATA"]=maxDownLinkData;
	  
	  	var primaryCgServerType=$("#primaryCgServerType").combobox("getValue");
	  	var primaryCgIpaddr = $("#primaryCgIpaddr").val();
	  	if(primaryCgServerType==1&&$.trim(primaryCgIpaddr).length!=0&&!checkIpAddressFormat("primaryCgIpaddr",primaryCgIpaddr)){
		  return;
	  	}
	  	$("#primaryCgIpaddrCheckSpan").text("");
	  	if(primaryCgServerType==2&&$.trim(primaryCgIpaddr).length!=0&&!isIPv6(primaryCgIpaddr)){
		  	$("#primaryCgIpaddrCheckSpan").text("<%=rb.getString("QingShuRuHeFaDeIPV6DiZhi")%>");//Please enter a valid IPv6 address.
		  	return;
	  	}
	  	params["PRIMARY_CG_SERVER_TYPE"]=primaryCgServerType;
	  	params["PRIMARY_CG_IPADDR"]=primaryCgIpaddr;
	  
	  	var secondaryCgServerType=$("#secondaryCgServerType").combobox("getValue");
	  	var secondaryCgIpaddr = $("#secondaryCgIpaddr").val();
	  	if(secondaryCgServerType==1&&$.trim(secondaryCgIpaddr).length!=0&&!checkIpAddressFormat("secondaryCgIpaddr",secondaryCgIpaddr)){
		  	return;
	  	}
	  	$("#secondaryCgIpaddrCheckSpan").text("");
	  	if(secondaryCgServerType==2&&$.trim(secondaryCgIpaddr).length!=0&&!isIPv6(secondaryCgIpaddr)){
		  	$("#secondaryCgIpaddrCheckSpan").text("<%=rb.getString("QingShuRuHeFaDeIPV6DiZhi")%>");
		  	return;
	  	}
	  	params["SECONDARY_CG_SERVER_TYPE"]=secondaryCgServerType;
	  	params["SECONDARY_CG_IPADDR"]=secondaryCgIpaddr;
	  	var apnGxEnable=$("#apnGxEnable").combobox("getValue");
	  	params["APN_GX_ENABLE"]=apnGxEnable;
	  	if(addr==""){
		  	showMsg('prompt_msg',"<%=rb.getString("QingXuanZheEPCFuWu")%>");
		  	$(".shuntChooseTit li").first().click();
		  	//$("#ueIpv4NetMaskCheckSpan").text("Plase select epc target server.");
		  	return;
	  	}
	  	savingCover();
	  	$.post("${ctx}/epc/configuration/addGwApnInfos.action", params, function(data){
		  	cancelSavingCover();
          	if (data["success"]) {
        	  	showMsg('success_msg',"<%=rb.getString("ChengGong")%>");
        	  	$("#apn_table").datagrid("reload");
        	  	$("#addApnSetting").slideUp(500);
        	  	$(".APNaddBtn .titleButtonText").html("<%=rb.getString("TianJia")%>");
   			    $(".APNaddBtn .addCircle").addClass("add_circle");
   			    $(".APNaddBtn .addCircle").removeClass("close_circle");
  			    showflag = true;
      		    //closeFn("addConfig","addCircle","operationTit","40px","operEpcConfiguration","addApnSetting","close",tianjia);
          	} else {
        	  	showMsg('error_msg',"<%=rb.getString("CaoZuoShiBai")%>");
        	  	//$("#ueIpv4NetMaskCheckSpan").text(data["message"]);
          	}
      	}, "json");
	}
    
    function checkApnNameConfig(epc_id,apnname){
    	var flag=0;
    	var request={};
    	request["EPC_ID"]=epc_id;
    	request["APN_NAME"]=apnname;
    	$.ajax({
			type: "post",
			url: "${ctx}/epc/configuration/checkApnNameConfig.action",
			data: request,
			async: false,
			dataType:"json",
			success: function(data) {
				if(data["message"]>0){
					flag=1;
				}
			}
		});
    	return flag;
    }
    
    //获取epc服务下apn配置个数
    function getApnConfigCount(epc_id){
    	var flag=0;
    	var request={};
    	request["EPC_ID"]=epc_id;
    	$.ajax({
			type: "post",
			url: "${ctx}/epc/configuration/getApnConfigCount.action",
			data: request,
			async: false,
			dataType:"json",
			success: function(data) {
				flag=data["message"];
			}
		});
    	return flag;
    }
    
    function checkIpPoolConfig(epc_id,apnname,ip){
    	var flag=0;
    	var request={};
    	ip = ip.replace(".","").replace(".","").replace(".","");
    	request["EPC_ID"]=epc_id;
    	request["APN_NAME"]=apnname;
    	request["IP"]=ip;
    	$.ajax({
			type: "post",
			url: "${ctx}/epc/configuration/checkIpPoolConfig.action",
			data: request,
			async: false,
			dataType:"json",
			success: function(data) {
				if(data["message"]>0){
					flag=1;
				}
			}
		});
    	return flag;
    }
    
    function checkIpPoolConfigCount(epc_id,apnname){
    	var flag=0;
    	var request={};
    	request["EPC_ID"]=epc_id;
    	request["APN_NAME"]=apnname;
    	request["IP"]="";
    	$.ajax({
			type: "post",
			url: "${ctx}/epc/configuration/checkIpPoolConfig.action",
			data: request,
			async: false,
			dataType:"json",
			success: function(data) {
				if(data["message"]>0){
					flag=data["message"];
				}
			}
		});
    	return flag;
    }
    
  	function updateApnConfiguration(){
	  var row=$("#apn_table").datagrid('getSelected');
	  //需要加判断 是否修改
	  if($("#m_ueStartIpv4Addr").val()==row.UE_START_IPV4_ADDR && $("#m_ueIpv4NetMask").val()==row.UE_IPV4_NET_MASK){
		  $(".modifyApnSetting").animate({right:'-1800px'},500);
		  return;
	  }
	  
	  var ueStartIpv4Addr=$("#m_ueStartIpv4Addr").val();
	  if(!checkIpAddressFormat("m_ueStartIpv4Addr",ueStartIpv4Addr)){
    	  return;
      }
	  var ueIpv4NetMask=$("#m_ueIpv4NetMask").val();
	  if(!checkMaskFormat("m_ueIpv4NetMask",ueIpv4NetMask)){
    	  return;
      }
	  savingCover();  
	  $.post("${ctx}/epc/configuration/updateGwApn.action", {APN_ID:row.APN_ID,APN_NAME:$("#m_gwApnName").val(),UE_START_IPV4_ADDR:$("#m_ueStartIpv4Addr").val(),UE_IPV4_NET_MASK:$("#m_ueIpv4NetMask").val(),ADDR:row.EPC_SERVER_IP,ADDR_PORT:row.EPC_PORT}, function(data){
		  cancelSavingCover();
          if (data["success"]) {
        	  showMsg('success_msg',"success");
        	  $("#apn_table").datagrid("reload");
        	  $(".modifyApnSetting").animate({right:'-1800px'},500);
          } else {
        	  $("#m_ueIpv4NetMaskCheckSpan").text(data["message"]);
          }
      }, "json");
  	}
  
  	function addGwApnIpPoolConfiguration(){
	  	 var selectValue = $("#APNchoseEPC").combobox("getValues");
		 var paramsArray = $("#APNchoseEPC").combobox("getData");
		 //选中的参数
		 var dataparams = paramsArray.filter(function(item){
		 		return item.EPC_ID == selectValue;
		 });
		 var params = {};
		 var addr=dataparams[0].IP;
		 var epc_port=dataparams[0].PORT; 
		 var epc_id=dataparams[0].EPC_ID; 
		 params["EPC_SERVER_IP"]=addr;
		 params["EPC_PORT"]=epc_port;
		 params["EPC_ID"]=epc_id; 

	  	var apnName=$("#apnName").val();
      	if(!checkRangLength("apnName",apnName,1,63)){
    	  	return;
      	}
      	if(!checkName("apnName",apnName)){
			return;
		}
      	/*
      	var reg = /^([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9_.-][a-zA-Z0-9]+)$/
      	$("#apnNameCheckSpan").text("");
  	  	if(!reg.test(apnName)){
  		  	$("#apnNameCheckSpan").text("<%=rb.getString("ZhiNengShuRuZiMuShuZiXiaHuaXianXiaoShuDian")%>");
  		  	return;
  	  	}*/
  	  	if(checkApnNameConfig(epc_id,apnName)<=0){
		  	$("#apnNameCheckSpan").text("<%=rb.getString("APNMingChengBuCunZai")%>");
		  	return;
  	  	}
  	  	if(checkIpPoolConfigCount(epc_id,apnName)>=32){
	  	  	$("#apnNameCheckSpan").text("<%=rb.getString("ZuiDaZhiChi32")%>");
		  	return;
  	  	}
      	params["APN_NAME"]=apnName;
      	var ipAddrAllocType=$("#ipAddrAllocType").combobox("getValue");
      	var ueStartIpv4Addr = $("#ueStartIpv4Addr").val();
      	var ueEndIpv4Addr = $("#ueEndIpv4Addr").val();
      	if((ipAddrAllocType==1 || ipAddrAllocType==3) && !checkIpAddressFormat("ueStartIpv4Addr",ueStartIpv4Addr)){
    	 	return;
      	}
      	if(ueStartIpv4Addr.length!=0 && checkIpPoolConfig(epc_id,apnName,ueStartIpv4Addr)>0){
    	  	$("#ueStartIpv4AddrCheckSpan").text("<%=rb.getString("UEStartIpv4AddrYiCunZai")%>");
		  	return;
      	}
      	if((ipAddrAllocType==1 || ipAddrAllocType==3) && !checkIpAddressFormat("ueEndIpv4Addr",ueEndIpv4Addr)){
    	  	return;
      	}
      	if(ueStartIpv4Addr.length!=0 && checkIpPoolConfig(epc_id,apnName,ueEndIpv4Addr)>0){
    	  	$("#ueEndIpv4AddrCheckSpan").text("<%=rb.getString("UE_End_Ipv4_AddrYiCunZai")%>");
		  	return;
      	}
      	var ipv4s = ueStartIpv4Addr.replace(".","").replace(".","").replace(".","");
      	var ipv4e = ueEndIpv4Addr.replace(".","").replace(".","").replace(".","");
      	if((ipAddrAllocType==1 || ipAddrAllocType==3) && parseInt(ipv4e,10)<parseInt(ipv4s,10)){
    	  	$("#ueEndIpv4AddrCheckSpan").text("<%=rb.getString("UEEndIpv4AddrBuNengXiaoYuUEStartIpv4Addr")%>");
		  	return;
      	}
      	params["IP_ADDR_ALLOC_TYPE"]=ipAddrAllocType;
      	params["UE_START_IPV4_ADDR"]=ueStartIpv4Addr;
      	params["UE_END_IPV4_ADDR"]=ueEndIpv4Addr;
      	var ueStartIpv6Addr = $("#ueStartIpv6Addr").val();
      	var ueEndIpv6Addr = $("#ueEndIpv6Addr").val();
      	$("#ueStartIpv6AddrCheckSpan,#ueEndIpv6AddrCheckSpan").text("");
      	if((ipAddrAllocType==2 || ipAddrAllocType==3) && !isIPv6(ueStartIpv6Addr)){
    	  	$("#ueStartIpv6AddrCheckSpan").text("<%=rb.getString("QingShuRuHeFaDeIPV6DiZhi")%>");
    	  	return;
      	}
      	if((ipAddrAllocType==2 || ipAddrAllocType==3) && !isIPv6(ueEndIpv6Addr)){
    	  	$("#ueEndIpv6AddrCheckSpan").text("<%=rb.getString("QingShuRuHeFaDeIPV6DiZhi")%>");
    	  	return;
      	}
      	params["UE_START_IPV6ADDR"]=ueStartIpv6Addr;
      	params["UE_END_IPV6ADDR"]=ueEndIpv6Addr;
      
      	var addrPrefixLength = $("#addrPrefixLength").val();
      	if(addrPrefixLength>128 && addrPrefixLength<1 ){
      		return;
      	}else{
      		params["ADDR_PREFIX_LENGTH"]=addrPrefixLength;
      	}
      	
	  	if(addr==""){
		  	showMsg('prompt_msg',"<%=rb.getString("QingXuanZheEPCFuWu")%>");
		  	$(".shuntChooseTit li").first().click();
		  	//$("#ueIpv4NetMaskCheckSpan").text("Plase select epc target server.");
		  	return;
	  	}
	  	savingCover();
	  	$.post("${ctx}/epc/configuration/addGwApnIpPoolInfos.action", params, function(data){
		  	cancelSavingCover();
          	if (data["success"]) {
        	  	showMsg('success_msg',"success");
        	  	$("#addIpPoolSetting").slideUp(500);
        	  	$(".APNIPaddBtn .titleButtonText").html("<%=rb.getString("TianJia")%>");
  			    $(".APNIPaddBtn .addCircle").addClass("add_circle");
  			    $(".APNIPaddBtn .addCircle").removeClass("close_circle");
  			    showIppoolFlag = true;
    		  	//closeFn("addIpPool","addCircle","operationTit","40px","addIpPoolConfig","addIpPoolSetting","close",tianjia);
        	  	//operEpcConfiguration("OFF","addApnSetting");
        	  	$("#apn_table_ipPool").datagrid("reload");
          	} else {
        	  	showMsg('error_msg',data["message"]);
        	  	$("#ueIpv4NetMaskCheckSpan").text(data["message"]);
          	}
      	}, "json");
  	}
  
  //校验子网掩码
  function checkMask(markIp){
	  var exp=/^(254|252|248|240|224|192|128|0)\.0\.0\.0|255\.(254|252|248|240|224|192|128|0)\.0\.0|255\.255\.(254|252|248|240|224|192|128|0)\.0|255\.255\.255\.(254|252|248|240|224|192|128|0)$/; 
	  return exp.test(markIp); 		
  }
  function checkMaskFormat(id,value){
	  if(!checkMask(value)){
	     //$("#"+id).focus().select();
	     $("#"+id+"CheckSpan").text("<%=rb.getString("QingShuRuHeFaDeYanMa")%>");
	     return false;
	  }else{
	     $("#"+id+"CheckSpan").text("");
	     return true;
	  }
  	}
  
  	//检查长度
  	function checkRangLength(id,value,minLength,maxLength){
      if(value=="" || value.length<minLength || value.length>maxLength){
		//$("#"+id).focus().select();
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
	
  	function checkIpAddressFormat(id,value){
	  if(!isValidIP(value)){
	     //$("#"+id).focus().select();
	     $("#"+id+"CheckSpan").text("<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>");
	     return false;
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
			//$("#"+id).focus().select();
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
  function impGwApnConfiguration(){
	  var selectValue = $("#APNchoseEPC").combobox("getValues");
	  if(selectValue[0] == ""){
			$('.prompt_msg').html("<%=rb.getString("QingXuanZheEPCFuWu")%>");
	  		var width = $('.prompt_msg').width();
	  		$('.prompt_msg').css("left",'50%');
	  		var left = parseFloat($('.prompt_msg').css("left")) - width/2;
	  		$('.prompt_msg').css("left",left+'px');
	  		$('.prompt_msg').animate({top:'55px'},200,function(){
	  			setTimeout(function(){
	  				$('.prompt_msg').animate({top:'-40px'},function(){
	  					$("#APNchoseEPC").combobox("showPanel");
	  				})
	  			},3000)
	  		})
			return;
		} 
	  var epc_id =  $("#APNchoseEPC").combobox('getValue');
		
	  var paramsArray = $("#APNchoseEPC").combobox("getData");
	  //选中的参数
	  var dataparams = paramsArray.filter(function(item){
	 		return item.EPC_ID == selectValue;
	  });
	  var epc_id = dataparams[0].EPC_ID;
	  var addr = dataparams[0].IP;
	  var epc_port = dataparams[0].PORT;
	 
	 <%--  if(addr==""||addr==undefined){
		  $.messager.alert(TiShi, "<%=rb.getString("QingXuanZheEPCFuWu")%>");
		  $("#APNStting .shuntChooseTit li").first().click();
		  return;
	  } --%>
	  if($("#filePath").val()==""){
		  $('#uploadFile').click();
		  return;
	  }
	  var filePath = $("#filePath").val();
	  if(!checkUploadFile(filePath)){
		  showMsg('prompt_msg',"file format error.");
		  return;
	  }
	  $("#ADDR").val(addr);
	  $("#ADDR_PORT").val(epc_port);
	  $("#EPC_ID").val(epc_id);
	  var param={};
	  param["EPC_ID"]=epc_id;
	  param["ADDR"]=addr;
      param["ADDR_PORT"]=epc_port;
	  savingCover();
	  $("#uploadForm_apn").form("submit", {
			dataType: 'json',
			success: function (data) {
				if(typeof data == 'string') data = eval("(" + data + ")");
				cancelSavingCover();
			    //alert(data["success"]+"==="+data["msg"]);
				if(data["success"] && data["msg"]=="0"){
					$("#failureText").text("<%=rb.getString("BuFenShuJuCuoWuQingChongXingBianJi")%>");
					$("#winDowloadFailureFile").window("setTitle", " <%=rb.getString("XinXi")%>");
					$("#winDowloadFailureFile").window("open");
					doSearchUrl('apn_table', param,'${ctx}/epc/configuration/getGwApnInfos.action');
				}else if(data["success"] && data["msg"]=="1"){
					showMsg('success_msg',"<%=rb.getString("ChengGong")%>");
					cancelGwApnConfiguration('importApnSetting')
				    doSearchUrl('apn_table', param,'${ctx}/epc/configuration/getGwApnInfos.action');
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
				var bool = checkParams(param);
				if(!bool) return false;
			}
	  });
  }
  function checkCdrFileDuration(){
	  var cdrFileDurationVal = parseInt($("#cdrFileDuration").val());
	  if(cdrFileDurationVal%60 == 0){
		  $("#cdrFileDurationCheckSpan").text("");
	  }else{
		  $("#cdrFileDurationCheckSpan").text("Please Enter a multiple of sixty");
	  }
  }
  function dowloadFailureFile(){
	  $("#downloadFailureApn").form("submit",{
		  onSubmit: function(param){
				var bool = checkParams(param);
				if(!bool) return false;
			}
	  });
	  $('#winDowloadFailureFile').window('close');
  }
  function checkAddrPreLength(){
	var addrPrefixLength = parseInt($("#addrPrefixLength").val());
	if(addrPrefixLength>128 || addrPrefixLength<1 ){
		$("#addrPrefixLengthCheckSpan").text("Int,min value:1,max value:128");
	}else{
		$("#addrPrefixLengthCheckSpan").text("")
	}
  }
  function impGwApnIpPoolConfiguration(){
	 
 	 var selectValue = $("#APNchoseEPC").combobox("getValues");
 	 if(selectValue[0] == ""){
			$('.prompt_msg').html("<%=rb.getString("QingXuanZheEPCFuWu")%>");
	  		var width = $('.prompt_msg').width();
	  		$('.prompt_msg').css("left",'50%');
	  		var left = parseFloat($('.prompt_msg').css("left")) - width/2;
	  		$('.prompt_msg').css("left",left+'px');
	  		$('.prompt_msg').animate({top:'55px'},200,function(){
	  			setTimeout(function(){
	  				$('.prompt_msg').animate({top:'-40px'},function(){
	  					$("#APNchoseEPC").combobox("showPanel");
	  				})
	  			},3000)
	  		})
			return;
		} 
	 var paramsArray = $("#APNchoseEPC").combobox("getData");
	 //选中的参数
	 var dataparams = paramsArray.filter(function(item){
	 		return item.EPC_ID == selectValue;
	 });
	 var epc_id = dataparams[0].EPC_ID;
	 var addr = dataparams[0].IP;
	 var epc_port = dataparams[0].PORT;
	 var param = {};
	  if($("#filePathIpPool").val()==""){
		  $('#uploadFileIP').click();
		  return;
	  }
<%-- 	  if(addr==""||addr==undefined){
		  $.messager.alert(TiShi, "<%=rb.getString("QingXuanZheEPCFuWu")%>");
		  $("#APNStting .shuntChooseTit li").first().click();
		  return;
	  } --%>
	  if($("#filePathIpPoolfilePathIpPool").val()==""){
		  $('#uploadFileIP').click();
		  return;
	  }
	  var filePath = $("#filePathIpPool").val();
	  if(!checkUploadFile(filePath)){
		  showMsg('prompt_msg',"<%=rb.getString("WenJianGeShiCuoWu")%>");
		  return;
	  }
	  $("#ADDR_IP").val(addr);
	  $("#ADDR_PORT_IP").val(epc_port);
	  $("#EPC_ID_IP").val(epc_id);
	  var param={};
		param["EPC_ID"]=epc_id;
		param["ADDR"]=addr;
		param["ADDR_PORT"]=epc_port;
	  savingCover();
	  $("#uploadForm_apn_ip_pool").form("submit", {
			dataType: 'json',
			success: function (data) {
				if(typeof data == 'string') data = eval("(" + data + ")");
				cancelSavingCover();
				if(data["success"] && data["msg"]=="0"){
					$("#failureText").text("<%=rb.getString("BuFenShuJuCuoWuQingChongXingBianJi")%>");
					$("#winDowloadFailureFile").window("setTitle", " <%=rb.getString("XinXi")%>");
					$("#winDowloadFailureFile").window("open");
					doSearchUrl('apn_table_ipPool', param,'${ctx}/epc/configuration/getGwApnIpPoolInfos.action');
				}else if(data["success"] && data["msg"]=="1"){
					showMsg('success_msg',"<%=rb.getString("CaoZuoChengGong")%>");
					doSearchUrl('apn_table_ipPool', param,'${ctx}/epc/configuration/getGwApnIpPoolInfos.action');
					cancelGwApnIpPoolConfiguration('importIpPoolSetting');
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
  
  function ColumnConfigAPN() {
		//itemList 传给后台用来加载排序选项的参数
		var itemList = "";
		var selContent = "";
		var checkList = $(".export1ConfigItem").find("input");
		$(".export1ConfigItem input[type='checkbox']").each(function() {
			itemList += ($(this).attr("item")+",");	
		});
		itemList = itemList.substring(0,itemList.length-1);
		//var selContent = "itemList";
		$(".export1ConfigItem input[type='checkbox']").each(function() {
			if (!this.checked) {
				if (selContent != "") {
					selContent += ",";
				}
				selContent += $(this).attr("item");
			}
		});
		columncellAPN = selContent;
		var sortcolumn = "";
		//选中的选项作为传递的参数
		$(".export1ConfigItem input[type='checkbox']").each(function() {
			if (this.checked) {
				if (sortcolumn != "") {
					sortcolumn += ",";
				}
				sortcolumn += $(this).attr("item");
			}
		});
		var params = {
			"hiddenColumn": selContent,
			"allColumnAPN": itemList
		};
		$.post("${ctx}/epc/configuration/epcColumnConfig.action", params, function (data) {
			$(".showHideItem").slideUp(500); 
			if (!data["success"]) {
				showMsg('error_msg',"<%=rb.getString("BaoCunShiBai")%>");
			} else {
				var field_arr = [];
				if("" != selContent){
					field_arr = selContent.split(",");
				}			
				//调整顺序后获取的数组（排序）
				sortcolumn = sortcolumn.toUpperCase();
			 	var sortColumnArr = sortcolumn.split(",");
				var loadtable =$.extend(true,[],apn_column);
				loadtable.splice(0,loadtable.length);
				loadtable.push({field:'APN_NAME',title:'<%=rb.getString("APNMingCheng")%>'});
				for(var i=0;i<sortColumnArr.length;i++){
					for(var j=0;j<apn_column.length;j++){
						if(apn_column[j].field == sortColumnArr[i]){
							loadtable.push(apn_column[j]);
							continue;
						}
					}
				} 
				var fitColumn = false;
			    var main_width = $("#gateWayRightDiv").width();
				if((main_width>1280 && field_arr.length>=5 )|| (main_width<=1280 && field_arr.length>=12)){
					fitColumn = true;
				}
				var tmpCols = [];
				$.each(loadtable,function(index,item){
					var col = $.extend({},item);
					tmpCols.push(col);
				});
				(function(fitCol,tmpCols){
					$('#apn_table').datagrid({fitColumns: true,columns:[tmpCols],onLoadSuccess:function(data){
						$(this).datagrid("fixRownumber");
				    	$(this).datagrid("enableContextmenuAutoSize");
						var cNames = $(this).datagrid('getColumnFields');
				          $.each(cNames,function(index,item){
				            var cOptions = $('#apn_table').datagrid('getColumnOption',item);
				            if(cNames.length<8){
				            	delete cOptions.fixed;
					            cOptions.auto = false;
					            cOptions.width = 10;
				            }else{
				            	cOptions.fixed = true;
				            	cOptions.auto = true;
					            delete cOptions.width;
				            }
				          });
						$(this).datagrid('fitColumns');
					}});
				})(fitColumn,tmpCols);
			}
		}, "json");	
	}
  
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
  		//var reg = /^([a-zA-Z0-9]|[a-zA-Z0-9]+|[a-zA-Z0-9]+[a-zA-Z0-9_.-]+[a-zA-Z0-9]+)$/
  		var reg = /^([a-zA-Z0-9]+(\.|-))*[a-zA-Z0-9]+$/
    	if(!reg.test(value)){
			$("#"+id).focus().select();
			$("#"+id+"CheckSpan").text("<%=rb.getString("ZhiNengShuRuZiMuShuZiXiaHuaXianXiaoShuDian")%>");
			return false;
	  	}else{
			$("#"+id+"CheckSpan").text("");
			return true;
	  	}
  	}
	
  	function chenkCDrInterTimer(){
  		var cdrInterimCdrTimer = $("#cdrInterimCdrTimer").val();
  		checkIntRange("cdrInterimCdrTimer",cdrInterimCdrTimer,1,120);
  	}
	//选择EPC
	function chooseShunt(){
		$("#enbMonitorSearchText").val("");
		$("#enbMonitorSearchTextIpPool").val("");
		$(".shuntChooseItem").slideUp(250);
		
		/* var tab=$("#tabs_service").tabs('getSelected');
		var index=$("#tabs_service").tabs('getTabIndex',tab);
		var pfs=$(ele).attr("pfs"); */
		var selectValue = $("#APNchoseEPC").combobox("getValues");
		var paramsArray = $("#APNchoseEPC").combobox("getData");
		//选中的参数
		var dataparams = paramsArray.filter(function(item){
				return item.EPC_ID == selectValue;
		});
		var param={};
		param["EPC_ID"]=dataparams[0].EPC_ID;
		param["ADDR"]=dataparams[0].IP;
		param["ADDR_PORT"]=dataparams[0].PORT;
		//selectedEPCItem 判断选择的是哪个epc的项目
		switch (selectedEPCItem){
			case 0:
				savingCover();
				 $.post("${ctx}/epc/configuration/addGwApn.action", param, function(data){
					 cancelSavingCover();
					 doSearchUrl('apn_table', param,'${ctx}/epc/configuration/getGwApnInfos.action');
				 }).error(function(){
					 cancelSavingCover();
					 showMsg('error_msg',"load error.");
				 });
			break;
			case 1:
				savingCover();
				 $.post("${ctx}/epc/configuration/addGwApnIpPool.action", param, function(data){
					 cancelSavingCover();
					 doSearchUrl('apn_table_ipPool', param,'${ctx}/epc/configuration/getGwApnIpPoolInfos.action');
				 }).error(function(){
					 cancelSavingCover();
					 showMsg('error_msg',"load error.");
				 });
			break
		
		}
	}  	
</script>
<style>
addr
ul.tabs{
    height:100%;
    border-right:1px solid #d1ecf5;
}

.tabs li a.tabs-inner{
    background-color:#fff !important;
    height:50px !important;
    line-height:50px !important;
    margin-top:0px;
    padding-left:0 !important;
}
.tabs li{
	margin:0 !important;
}

.tabs li.tabs-selected  a.tabs-inner{
    background-color:#daeef9 !important;
}
.tabs  a.tabs-inner:hover{
    background-color:#f2fbff !important;
}
 
.tabs-header{
    padding:0;
}

.tabs-header .tabs-header-left .tabs{
    padding:0;
    border-top-width:0;
    border-bottom-width:1px;
    border-left-width:1px;
    boder-right-width:10px solid;
    width: 249px;
}
.tabs-header,.tabs-tool{
    background-color:#fff;
}

.btn_cen a{
	margin:12px 20px 0 0 ;
	float:right;
}
.nTab {
    float: left;
    margin: 0 auto;          
    background-position: left;
    background-repeat: repeat-y;
    margin-bottom: 2px;
    width:100%;
}

 .nTab .TabTitle {     
    clear: both;
    height: 41px;
    overflow: hidden;
    border-bottom: 1px #C7C7CD solid; 
}

.nTab .TabTitle ul {
    margin: 0;
    padding: 0;
}

.nTab .TabTitle li {
    float: left;
    width: 150px;
    height: 40px;
    cursor: pointer;
    padding-top: 6px;
    text-align: center;
    padding-bottom: 1px;
    list-style-type: none;
}

.nTab .TabTitle .active {

    border-bottom: 3px #0095e1 solid;
}

.nTab .TabTitle .normal {
 
}

.nTab .TabContent {
    width: auto;
    background: #fff;
    margin: 0px auto;
    padding: 30px 0 0 0;               
}

.back_hover:hover{
	background:#9bd8fa !important;
}
.planshow{
	margin:0 0 10px 0 !important;
}
#tabs_service ul.tabs{
	margin-top:0;
	width: 249px;
}
#tabs_service .tabs-header{
	width: 249px;
}
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
.epcMgtHeader{
	padding:20px;
	padding-right:0px;
	padding-bottom:10px;
	position:relative;
	background:#FFFFFF;
	z-index:20;
	height: 60px;
	width: 95%;
}
.epcTit{
	height:34px;
	border-bottom:2px solid #F4FAFD;
	padding-left:20px;
	box-sizing:border-box;
	margin-right:95px;
}
.epcTit li,.newEGWTit li{
	float:left;
	width:150px;
	text-align:center;
	display:inline-block;
	height:34px;
	font-size:16px;
	color:#c2c2c2;
	margin-right:20px;
	cursor:pointer;
	border-bottom: 2px solid #8DCAF9;
    box-sizing: border-box;
}
.addConfig,.addIpPool{
/* 	position:absolute;
	width:42px;
	height:42px;
	background:#339fd9;
	border-radius:25px;
	top:30px;
	right:35px;
	box-shadow:3px 5px 17px rgba(51,153,204,0.3); */
}
.addConfigImport,.importIpPool{
/* 	position:absolute;
	width:42px;
	height:42px;
	background:#339fd9;
	border-radius:25px;
	top:30px; */
	right:-28px;
	/* box-shadow:3px 5px 17px rgba(51,153,204,0.3); */
}
.tabs li.tabs-selected a.tabs-inner{
  font-weight: normal;
}
.tabs-header-left .tabs{
  padding: 0px 0 0 2px;
}
.epcConfigInfoItemDiv{
	display:inline-block;
	width:400px;
	margin:5px 60px 0 20px;
	vertical-align:top;
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
	line-height:25px;
	height:25px;
	color: red;
}
.addTrafficSetting ,.addApnSetting,.addIpPoolSetting,.importIpPoolSetting{
	/* height:95%; */
	 /* width:94.5%; */ 
	position:absolute;
	display:none;
	top:40px; 
	bottom:0px;
	right:0px;
	z-index:10;
	background:white;
	padding: 0px 0px 0px 60px;
	overflow-y:auto
}
.modifyTrafficSetting ,.modifyApnSetting{
	background:#FFFFFF;
	position:absolute;
	z-index:200;
	height:440px;
	right:-1800px;
	width:500px;
	top:2px;
	box-shadow:0 0 50px rgba(158,200,222,0.35);	
}
.shuntChoose{
	position:relative;
	margin-bottom:20px;
}
.shuntChooseTit li{
	cursor:pointer;
}
.secondTitle{
	margin-top:0px;
	margin-bottom:5px;
	height:34px;
	border-bottom:2px solid #f7fafd;
	box-sizing:border-box;
}
.secondTitle li{
	line-height:34px;
	width:170px;
	float:left;
	padding:0 30px;
	text-align:center;
	display:inline-block;
	height:34px;
	font-size:14px;
	color:#c2c2c2;
	margin-right:20px;
	margin-left:20px;
	cursor:default;
	border-bottom:2px solid #b0e1f6;
	box-sizing:border-box;
	color:#92c8df;
}
.chooseArrow{
	display:inline-block;
	margin-left:15px;
}
.shuntChooseItem{
	border:1px solid #d1ecf5;
	width:170px;
	position:absolute;
	margin-left:20px;
	top:34px;
	background:white;
	display:none;
	z-index:100;
}

.shuntChooseItem li{
	height:26px;
	line-height:26px;
	text-align:center;
	cursor:pointer;
	border-bottom:1px solid #d1ecf5;
}
.shuntChooseItem li:hover{
	background:#e1f2fa;
}
.shuntChooseItem li:active{
	background:#c4e6f5;
}
fieldset {
	padding: 6px;
	margin: 0px 0px;
	width: 792px;
	color: #333;
	border: #c9d1d6 solid 1px;
}
fieldset input{
	margin:0 10px;
	vertical-align:middle;
}
legend{
	padding:0 10px;
}
.addToeGW{
	-webkit-transform:rotate(45deg);
	-moz-transform:rotate(45deg);
	-ms-transform:rotate(45deg);
	-o-transform:rotate(45deg);
	transform:rotate(45deg);
}
.newEGWTit{
	height:60px;
	line-height:60px;
	margin:20px 57px;
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
/*显示隐藏列*/
.showHideItem{
	position:absolute;
	width:364px;
/* 	height:560px; */
	background:white;
	z-index:888;
	padding:0px 3px 0px;
	left:0px;
	top:30px;
	display:none;
	box-shadow:5px 10px 23px 0px rgba(201,212,231,0.50);
}
.showHideItem input{
	margin-top:-2px;
	margin-bottom:1px;
	vertical-align:middle;
	margin-right:20px;
}
.selectAll{
	height:32px;
	width:334px;
	padding:28px 0px 0px 30px;		
}
.sortul{
	width:364px;
	height:35vh; 
	padding:0px 0px 0px 0px;
	overflow:auto;
	overflow-x:hidden;
}
.sortul .export1ConfigItem{
	width:334px;
	height:30px;
	margin-top:2px;
	padding-left:30px;	
	line-height:30px;
}
.sortul .export1ConfigItem span{
	display:block;
	width:20px;
	height:20px;
	float:right;
	margin-right:20px;
	margin-top:5px;
}
.sortul .export1ConfigItem span.movesTop:hover{
	cursor:pointer;
}
.handleMouseDown{
	box-shadow:2px 2px 0 0 #C6C6C6;
}
.tabs-title{
	margin-left:10px;
}
/* 搜索框的样式 */
/* .queryConfig input{
	font-size:12px;
	width:300px;
	height:33px;
	border:none;
	border-bottom:1px solid #D0D9DE;
	-webkit-box-sizing:border-box;
	-moz-box-sizing:border-box;
	box-sizing:border-box;
	margin-left:20px;
	vertical-align:bottom;
}
.queryConfig img{
	cursor:pointer;
	vertical-align:bottom;
} */
.titleButtonText{
}
/* EPC新样式 */
.EPCMainPageCointer{
	position:relative;
	width:100%;
	height:100%; 
	/* min-width:800px;
	min-height:760px; */
}
/* 新样式 */
.pitchOnItem{
	color:#000000 !important;
}
.EPCLeftContainer{
	height:100%;
	width:280px;
	float:left;
	background:#FFFFFF;
}
.EPCLeftContainer h3{
	font-weight:normal;
	font-size:17px;
	color:#7993B6;
	margin:20px 0px 20px 20px;
}
.EPCLeftContainer .PCRFList{
	height:160px;
	width:100%;
	margin-top:20px;
}
.EPCLeftContainer .PCRFList li{
	height:40px;
	width:calc(100% - 36px);
	padding-left:36px;
	line-height:40px;
	font-size:12px;
	color:#94B0D5;
}
.EPCLeftContainer .PCRFList li:hover{
	background:#E3F3FB;
}
.EPCRightContainer{
	position:absolute;
	top:0;
	bottom:0;
	left:295px;
	right:0;
	background:#FFFFFF;
	overflow:hidden;
}
.EPCRightContainer .EPCRightItem{
	position:absolute;
	left:0;
	right:0;
	top:0;
	bottom:0;
	z-index:9;
	display:none;
	background:#FFFFFF;
}
.zIndex{
	z-index:999 !important;
}
</style>