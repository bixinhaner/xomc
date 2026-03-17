<%@ page import="java.util.Locale"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<script type="text/javascript">
	var ctx = "${ctx}";
</script>

<style>
.epcConfigInfoItemDiv{
	display:inline-block;
	width:471px;
	margin:5px 0px 0 0;
	vertical-align:top;
	margin-left:0px;
}
.epcConfigTrafficDiv{
	display:inline-block;
	width:471px;
	margin:8px 0px 0 0;
	vertical-align:top;
}
.epcConfigInfoItemDiv label{
	display:block;
	line-height:25px;
	color:#797979;
}
.epcConfigInfoItemDiv input{
	width:396px !important;
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
.epcConfigTrafficDiv label{
	display:block;
	line-height:25px;
	color:#797979;
}
.epcConfigTrafficDiv input{
	width:396px !important;
}
.epcConfigTrafficDiv img{
	margin-left:10px;
}
.epcConfigTrafficDiv .prompt{
	display:block;
	line-height:25px;
	height:25px;
	color: red;
}
.addTrafficSetting ,.addFilterSetting ,.addImsiTrafficSetting ,.addApnTrafficSetting{
	position:absolute;
	display:none;
	top:30px;
	bottom:0px;
	z-index:10;
	left:0px;
	right:0px;
	background:#FFFFFF;
	padding:0px 60px 0px;
	overflow:auto
}
.modifyTrafficSetting ,.modifyFilterSetting,.modifyImsiTrafficSetting,.modifyApnTrafficSetting{
	background:#FFFFFF;
	position:absolute;
	z-index:84;
	display:none;
	left:0px;
	right:0px;
	top:41px;
	bottom:0px;
	overflow-y:auto;
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
	background:#FFFFFF;
	display:none;
	z-index:100;
}
.shuntChooseItem li{
	border-bottom:1px solid #d1ecf5;
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
.submitConfig{
	margin-top:50px;
	margin-left:20px;
}
.epcConfigInfoItem{	
	height:calc(100% - 120px);
	padding-left:60px;
	padding-right:0px;
	margin:0px 0 30px;
}
.addImsiTrafficSetting .selectedImsiDiv .datagrid-body table{
	width: 100%;
}
.modifyImsiTrafficSetting .selectedImsiDiv .datagrid-body table{
	width:100%;
}
.addApnTrafficSetting .selectedApnDiv .datagrid-body table{
	width:100%;
}
.addTrafficSetting .selectedRuleDiv .datagrid-body table{
	width:100%;
}
.modifyApnTrafficSetting .selectedApnDiv .datagrid-body table{
	width:100%;
}
.modifyTrafficSetting .selectedRuleDiv .datagrid-body table{
	width:100%;
}
.EPCMainPageCointer{
	position:relative;
	width:100%;
	height:100%;
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
	cursor:pointer;
}
.EPCLeftContainer .PCRFList li:hover{
	background:#E3F3FB;
}
.EPCLeftContainer .PCRFList li:active{
	background:#DAEEF9;
}
.EPCRightContainer{
	position:absolute;
	top:0;
	bottom:0;
	left:295px;
	right:0;
	background:#FFFFFF;
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
.modifyContainer{
	display:flex;
	width:870px;
}
.leftTableDiv{
	flex: 5 1 50%;
	height:500px;	
}
.centerBtnGroup{
	flex: 1 0 auto;
	height:500px;
	width:80px;	
}
.rightTableDiv{
	flex: 5 1 50%;
	height:443px;
}
.tabsTitle{
	border:none;
}
</style>
<div class="EPCMainPageCointer">
	<!-- 左侧导航 -->
	<div class="EPCLeftContainer">
		<div class="tabsTitle"><span class='active'><%=rb.getString("PCCPeiZhiXiang")%></span></div>
		<!-- 选择EPC -->
		<div style="margin:20px 0 0 36px;">
			<select id = "PCCchoseEPC" class="easyui-combobox border border-box combobox-f combo-f textbox-f" data-options="editable:false" name="software_version" style="padding-top:0px;height:26px;width:200px;z-index:999"></select>
		</div>
		<!-- PCRF配置列表 -->
		<ul id="PCRFList" class="PCRFList">
			<li class="pitchOnItem">PCC Policy Function</li>
			<li>PCC Rule</li>
			<li>IMSI Configuration</li>
			<li>APN Configuration</li>
		</ul>
	</div>
		
	<!-- 右侧详细 -->
	<div class="EPCRightContainer">
		
		<!-- PCC Policy Function -->
		<div class="EPCRightItem" style="display:block;">
        	<div class="tabsTitle"><span class='active'>PCC Policy Function</span></div>
			<!-- 右上角添加按钮 -->
			<div class="circleIcon" id="addeGWFilter" myOpType="1" onclick="operEpcConfiguration('ON','addFilterSetting')">
				<span class="el-icon el-icon-circle-add"></span>
				<div class="titleButtonText"><%=rb.getString("TianJia")%></div>
			</div>
	        <!-- 搜索框 -->
	        <div id="queryPacketFilterStting" class="toolbarContainer" style="margin-left:15px;">
		  		<div class="queryGroup">
		  			<input id="sprPacketFilter" placeholder="<%=rb.getString("APPMingCheng")%>"/>
		  			<b class="el-icon el-icon-common-search" onclick="queryPacketFilter()"></b>
		  		</div>
			</div>   
			<!-- 表格 -->
			<div class="contentDiv">
			   	<table class="easyui-datagrid" id="filter_table" fit="true" data-options="fitColumns:true,singleSelect:true,border:false,toolbar:'#queryPacketFilterStting',
			                    rownumbers:true,url:'${ctx}/epc/configuration/getFilterInfos.action?EPC_ID=-1',pageSize:${pageSize},pageList:${pageList},striped:true,
			                    pagination:true,pagePosition:'bottom',onLoadSuccess: datagridLoadSuccess,
			                    onLoadError:datagridLoadError">
					<thead>
						<tr>
						    <th data-options="field:'NAME'" width="15"><%=rb.getString("EPCMingChen")%></th>
							<th data-options="field:'PF_ID',sortable:true" width="10">PF ID</th>
							<th data-options="field:'APP_NAME',sortable:true" width="20"><%=rb.getString("APPMingCheng")%></th>
							<th data-options="field:'PROTOCOL_TYPE',sortable:true" width="10"><%=rb.getString("XieYi")%></th>
							<th data-options="field:'PROTOCOL'" hidden="true" width="1">PROTOCOL</th>
							<th data-options="field:'IP_MASK',sortable:true"  width="20"><%=rb.getString("IPYanMa")%></th>
							<th data-options="field:'PORT',sortable:true"  width="10"><%=rb.getString("DuanKou")%></th>
							<th data-options="field:'operation',formatter: operFormatterFilter,fixed:true" width="80"><%=rb.getString("CaoZuo")%></th>
							<th data-options="field:'EPC_SERVER_IP'" hidden="true" width="1"></th>
							<th data-options="field:'EPC_PORT'" hidden="true" width="1"></th>
							<th data-options="field:'EPC_ID'" hidden="true" width="1"></th>
							<th data-options="field:'ID'" hidden="true" width="1"></th>
						</tr>
					</thead>
			  	</table>
			</div> 
			
			<!-- 增加过滤PCC  -->
			<div class="addFilterSetting" style="display:none">
			  	<div class="epcConfigInfoItemDiv">
				    <label for="protocol"><%=rb.getString("XieYi")%></label>
				    <select id="protocol" class="easyui-combobox border border-box" style="height:27px;width:395px;" data-options="editable:false"></select>
				    <span id="protocolCheckSpan" class="prompt" ></span>
				</div>
			  	<div class="epcConfigInfoItemDiv">
			    	<label for="filterAPP_NAME"><%=rb.getString("APPMingCheng")%></label>
			    	<input id="filterAPP_NAME" type="text" value="" maxlength="64"  class="easyui-validatebox border border-box" style="width:360px;" onblur="if(checkRangLength(this.id,this.value,1,64)){checkName(this.id,this.value);}"/>
			    	<span id="filterAPP_NAMECheckSpan" class="prompt" ></span>
			  	</div>
			  	<div class="epcConfigInfoItemDiv">
			    	<label for="filterIpMask"><%=rb.getString("IPYanMa")%> (example:192.168.9.20/24)</label>
			    	<input id="filterIpMask" type="text" value="" maxlength="20" class="easyui-validatebox border border-box" style="width:360px;"/>
			    	<span id="filterIpMaskCheckSpan" class="prompt" ></span>
			  	</div>
			  	<div class="epcConfigInfoItemDiv">
			    	<label for="filterPort"><%=rb.getString("DuanKou")%> (1~65535)</label>
			    	<input id="filterPort" type="text" value="" maxlength="11" class="easyui-validatebox border border-box" style="width:360px;" onblur="checkPortOrRange(this.id,this.value)" />
			    	<span id="filterPortCheckSpan" class="prompt" ></span>
			  	</div>
			  	<div class="submitConfig" style="margin-top:20px;margin-bottom:20px;">
		    	  	<a href="#" class="linkbutton" style="" onclick="addFilterConfiguration()"><span><%=rb.getString("BaoCun")%></span></a>
		  	  	</div>
			</div>
			
			<!-- 编辑pcc policy function -->
			<div class="modifyFilterSetting">
			  	<div class="newEGWTit">				
		         	<div id="filterCpeName" style="display:none;color:#7993B6;font-size:16px;"></div>
			  	</div>
			  	<div class="epcConfigInfoItem">
				  	<div class="epcConfigInfoItemDiv" style="margin-left:0px;">
				    	<label for="m_pfID">PF ID</label>
				    	<input id="m_pfID" type="text" value=""  class="easyui-validatebox border border-box" style="width:360px;" onblur="" disabled="disabled"/>
				    		<span id="m_pfIDCheckSpan" class="prompt" ></span>
				  	</div>
				  	<div class="epcConfigInfoItemDiv" style="margin-left:0px;">
				    	<label for="m_protocol"><%=rb.getString("XieYi")%></label>
				    	<select id="m_protocol" class="easyui-combobox border border-box" style="height:27px;width:360px;" data-options="editable:false"></select>
				    	<span id="m_protocolCheckSpan" class="prompt" ></span>
				  	</div>
				  	<div class="epcConfigInfoItemDiv" style="margin-left:0px;">
				    	<label for="m_filterAPP_NAME"><%=rb.getString("APPMingCheng")%></label>
				    	<input id="m_filterAPP_NAME" type="text" value="" maxlength="64" class="easyui-validatebox border border-box" style="width:360px;" onblur="if(checkRangLength(this.id,this.value,1,64)){checkName(this.id,this.value);}"/>
				    	<span id="m_filterAPP_NAMECheckSpan" class="prompt" ></span>
				  	</div>
				  	<div class="epcConfigInfoItemDiv" style="margin-left:0px;">
				    	<label for="m_filterIpMask"><%=rb.getString("IPYanMa")%> (example:192.168.9.20/24)</label>
				    	<input id="m_filterIpMask" type="text" value=""  class="easyui-validatebox border border-box" style="width:360px;" />
				    	<span id="m_filterIpMaskCheckSpan" class="prompt" ></span>
				  	</div>
				  	<div class="epcConfigInfoItemDiv" style="margin-left:0px;">
				    	<label for="m_filterPort"><%=rb.getString("DuanKou")%> (1~65535)</label>
				    	<input id="m_filterPort" type="text" value="" maxlength="11" class="easyui-validatebox border border-box" style="width:360px;" onblur="checkPortOrRange(this.id,this.value)" />
				    	<span id="m_filterPortCheckSpan" class="prompt" ></span>
				  	</div>
				  	<div style="margin-top:20px;">
						<a onclick="updateFilterConfiguration()" class="linkbutton"><span><%=rb.getString("BaoCun")%></span></a> 
				  	</div>
			  	</div>
			</div>			
		</div>
		
		<!-- PCC Rule -->
		<div class="EPCRightItem">
			<div class="singleTitle">PCC Rule</div>
			<!-- 右上角添加按钮 -->
			<div class="circleIcon" id="addeGWTraffic" myOpType="1" onclick="operEpcConfiguration('ON','addTrafficSetting')">
				<span class="el-icon el-icon-circle-add" ></span>
				<div class="titleButtonText"><%=rb.getString("TianJia")%></div>
			</div>
	        <!-- 搜索框 -->
	        <div id="queryPCCRULEFilterStting" class="toolbarContainer" style="margin-left:15px;">
		  		<div class="queryGroup">
		  			<input id="sprCommonTraffic" placeholder="<%=rb.getString("PCCMingCheng")%>"/>
		  			<b class="el-icon el-icon-common-search" onclick="queryPCCRULEFilter()"></b>
		  		</div>
			</div>   
		
			<!-- 表格 -->
			<div class="contentDiv">
				<table class="easyui-datagrid" id="traffic_table" fit="true" data-options="fitColumns:true,singleSelect:true,border:false,toolbar:'#queryPCCRULEFilterStting',
		                    rownumbers:true,url:'${ctx}/epc/configuration/getTrafficInfos.action?EPC_ID=-1',pageSize:${pageSize},pageList:${pageList},striped:true,
		                    pagination:true,pagePosition:'bottom',onLoadSuccess: datagridLoadSuccess,
		                    onLoadError:datagridLoadError">
					<thead>
						<tr>
						    <th data-options="field:'EPC_NAME'" width="120"><%=rb.getString("EPCMingChen")%></th>
							<th data-options="field:'PCC_NAME',sortable:true" width="120"><%=rb.getString("PCCMingCheng")%></th>
							<th data-options="field:'QCI',sortable:true" width="90">QCI</th>
							<th data-options="field:'ARP_PL',sortable:true"  width="90">ARP PL</th>
							<th data-options="field:'ARP_PCI',formatter: getDisabledFormatter,sortable:true"  width="90">ARP PCI</th>
							<th data-options="field:'ARP_PVI',formatter: getDisabledFormatter,sortable:true"  width="90">ARP PVI</th>
							<th data-options="field:'MBR_UL',sortable:true" width="90">MBR UL</th>
							<th data-options="field:'MBR_DL',sortable:true" width="90">MBR DL</th>
							<th data-options="field:'GBR_UL',sortable:true" width="90">GBR UL</th>
							<th data-options="field:'GBR_DL',sortable:true" width="90">GBR DL</th>
							<th data-options="field:'PRECEDENCE',sortable:true" width="100">PRECEDENCE</th>
							<th data-options="field:'PF_LIST',formatter: getPF_ListFormatter" width="180">PF LIST</th>
							<th data-options="field:'operation',formatter: operFormatter,fixed:true" width="80"><%=rb.getString("CaoZuo")%></th>
							<th data-options="field:'EPC_SERVER_IP'" hidden="true" ></th>
							<th data-options="field:'EPC_PORT'" hidden="true" ></th>
							<th data-options="field:'EPC_ID'" hidden="true"></th>
							<th data-options="field:'ID'" hidden="true"></th>
						</tr>
					</thead>
			  	</table>
			</div>
				
			<!-- 增加 PCC Rule -->
		   	<div class="addTrafficSetting" >
				<div id="addTrafficSetting_content" style="overflow-y:auto;">
					<div class="epcConfigInfoItemDiv">
					   	<label for="trafficName"><%=rb.getString("PCCMingCheng")%></label>
					   	<input id="Traffic_PCC_NAME" type="text" value="" maxlength="40" class="easyui-validatebox border border-box" style="width:300px;" onblur="if(checkRangLength(this.id,this.value,1,40)){checkName(this.id,this.value);}"/>
					  	<span id="Traffic_PCC_NAMECheckSpan" class="prompt" ></span>
					</div>
					<div class="epcConfigInfoItemDiv">
					  	<label for="qci">QCI</label>
					  	<input id="qci" type="text" value="" maxlength="1" onkeyup="value=value.replace(/[^\d]/g,'');setGBR(this.value)" class="easyui-validatebox border border-box" style="width:300px;" onblur="checkIntRange(this.id,this.value,1,9)" />
					  	<span id="qciCheckSpan" class="prompt" ></span>
					</div>
					<div class="epcConfigInfoItemDiv">
					  	<label for="ARP_PL">ARP PL</label>
					  	<input id="ARP_PL" type="text" value="" maxlength="2" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" style="width:300px;" onblur="checkIntRange(this.id,this.value,1,15)" />
					  	<span id="ARP_PLCheckSpan" class="prompt" ></span>
					</div>
					<div class="epcConfigInfoItemDiv">
					  	<label for="ARP_PCI">ARP PCI</label>
					  	<select id="ARP_PCI" class="easyui-combobox border border-box" style="height:27px;width:396px;" data-options="editable:false">
					      	<option value="0">disabled</option>
					      	<option value="1">enable</option>
					  	</select>
					 	<span id="ARP_PCICheckSpan" class="prompt" ></span>
					</div>
					<div class="epcConfigInfoItemDiv">
					  	<label for="ARP_PVI">ARP PVI</label>
					  	<select id="ARP_PVI" class="easyui-combobox border border-box" style="height:27px;width:396px;" data-options="editable:false">
					      	<option value="0">disabled</option>
					      	<option value="1">enable</option>
					  	</select>
					  	<span id="ARP_PVICheckSpan" class="prompt" ></span>
					</div>
					<div class="epcConfigInfoItemDiv">
					  	<label for="PRECEDENCE">PRECEDENCE</label>
					  	<input id="PRECEDENCE" type="text" value="" maxlength="3" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" style="width:300px;" onblur="checkIntRange(this.id,this.value,0,255)" />
					  	<span id="PRECEDENCECheckSpan" class="prompt" ></span>
					</div>
					<div class="epcConfigInfoItemDiv">
					  	<label for="mbrUL">MBR UL (bps:1000~104857600)</label>
					  	<input id="mbrUL" type="text" value="" maxlength="9" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" style="width:300px;" onblur="checkIntRange(this.id,this.value,1000,104857600)" />
					  	<span id="mbrULCheckSpan" class="prompt" ></span>
					</div>
					<div class="epcConfigInfoItemDiv">
					  	<label for="mbrDL">MBR DL (bps:1000~104857600)</label>
					  	<input id="mbrDL" type="text" value="" maxlength="9" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" style="width:300px;" onblur="checkIntRange(this.id,this.value,1000,104857600)" />
					  	<span id="mbrDLCheckSpan" class="prompt" ></span>
					</div>
					<div class="epcConfigInfoItemDiv">
					  	<label for="gbrUL">GBR UL (bps:1000~104857600)</label>
					  	<input id="gbrUL" type="text" value="" maxlength="9" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" style="width:300px;"/>
					  	<span id="gbrULCheckSpan" class="prompt" ></span>
					</div>
					<div class="epcConfigInfoItemDiv">
					  	<label for="gbrDL">GBR DL (bps:1000~104857600)</label>
					  	<input id="gbrDL" type="text" value="" maxlength="9" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" style="width:300px;"/>
					  	<span id="gbrDLCheckSpan" class="prompt" ></span>
					</div>
						
					<!-- 表格  -- 选择PCC Rule -->
					<div>
					  	<%-- <div class="singleTitle" >PCC Rule <%=rb.getString("XuanZe")%></div> --%>
					  	<ul class="omcPageTitleContainer" style="padding-left:0px;">
							<li class="default">PCC Rule<%=rb.getString("XuanZe")%></li>
						</ul>
					  	<div class="modifyContainer">
					      	<div class="leftTableDiv">
						      	<!-- 工具栏 修改PCC Rule -->
						  		<div style="margin-top:10px;margin-bottom:15px;">
						  			<p><%=rb.getString("QuanBuLieBiao")%></p>
						  			<div class='queryGroup' style='margin-left:0px;'>
						  				<input id="query_pcrf_pf_id" placeholder="PF ID" style="width:200px;"/>
										<b class="el-icon el-icon-common-search" onclick="serialPCRFPF_ID()"></b>
						  			</div>
						  		</div>
										
								<div style="height:400px;">
									<table class="easyui-datagrid" id="pcc_rule_table_list" fit="true" data-options="fitColumns:true,singleSelect:false,border:true,
						                    rownumbers:true,url:'${ctx}/epc/configuration/getFilterInfos.action?EPC_ID=-1',pageSize:${pageSize},pageList:${pageList},striped:true,
						                    pagination:true,pagePosition:'bottom',onBeforeLoad:onBeforeladtaskTable,
						                    onLoadError:datagridLoadError">
										<thead>
											<tr>
												<th data-options="field:'PF_ID'" width="80">PF ID</th>
												<th data-options="field:'APP_NAME'" width="150"><%=rb.getString("APPMingCheng")%></th>
											</tr>
										</thead>
								    </table> 
								</div>  	
					      	</div>
							<div class="centerBtnGroup" style="width: 80px;">
								<a href="javascript:void(0);" class="arrow_right" style="margin-top:110px" onclick="addSelectedCell_auto_confirm('pcc_rule_table_list','selectedRule_auto_confirm')"></a>
								<a href="javascript:void(0);" class="arrow_left" style="margin-top: 10px;" onclick="delSelectedCell_auto_confirm('selectedRule_auto_confirm')"></a>
						    </div>
					      	<div class="rightTableDiv">
					      	 	<p style="height:72px;line-height:35px;"><%=rb.getString("YiXuanLieBiao")%></p>
					     	 	<div style="height:400px;">
						      	 	<table class="easyui-datagrid" id="selectedRule_auto_confirm" fit="true" data-options="fitColumns:true,singleSelect:false,border:true,
						                    rownumbers:true,url:'',pageSize:${pageSize},pageList:${pageList},striped:true,data:[],
						                    pagination:true,pagePosition:'bottom',toolbar:'',onLoadSuccess:loadSuccess,
						                    onLoadError:datagridLoadError">
										<thead>
											<tr>
												<th data-options="field:'PF_ID'" width="80">PF ID</th>
												<%--<th data-options="field:'APP_NAME'" width="150"><%=rb.getString("APPMingCheng")%></th>--%>
											</tr>
										</thead>
								    </table>
								</div>
					      	</div>
					  	</div>
					</div>
					<div style="margin-top:20px;margin-left:20px;margin-bottom:20px;height:30px;">
				    	<a href="#" class="linkbutton" style="float:left;" onclick="addTrafficConfiguration()"><span><%=rb.getString("BaoCun")%></span></a>
				  	</div>
				 </div>
			</div>
			
			<!-- 修改 PCC Rule -->
			<div class="modifyTrafficSetting" style="left:0px;right:0px;top:41px;bottom:0px;padding-bottom:20px;overflow:auto">
				<div class="newEGWTit">				
			    	<div id="trafficCpeName" style="display:none;color:#7993B6;font-size:16px;"></div>
			    </div>
			    <div class="epcConfigInfoItem">
					<div class="epcConfigTrafficDiv">
					    <label for="PCC_NAME"><%=rb.getString("PCCMingCheng")%></label>
					    <input id="m_PCC_NAME"  type="text" value="" disabled="disabled" maxlength="40" class="easyui-validatebox border border-box" style="width:300px;" onblur="if(checkRangLength(this.id,this.value,1,40)){checkName(this.id,this.value);}" />
						<span id="m_PCC_NAMECheckSpan" class="prompt" ></span>
					</div>
					<div class="epcConfigTrafficDiv">
					    <label for="qci">QCI</label>
					    <input id="m_qci" type="text" value="" maxlength="1" onkeyup="value=value.replace(/[^\d]/g,'');setGBR_m(this.value)" class="easyui-validatebox border border-box" style="width:300px;" onblur="checkIntRange(this.id,this.value,1,9)" />
					    <span id="m_qciCheckSpan" class="prompt" ></span>
					</div>
					<div class="epcConfigTrafficDiv">
						<label for="mARP_PL">ARP PL</label>
						<input id="m_ARP_PL" type="text" value="" maxlength="2" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" style="width:300px;" onblur="checkIntRange(this.id,this.value,1,15)" />
						<span id="m_ARP_PLCheckSpan" class="prompt" ></span>
					</div>
					<div class="epcConfigTrafficDiv">
						<label for="mARP_PCI">ARP PCI</label>
						<select id="m_ARP_PCI" class="easyui-combobox border border-box" style="height:27px;width:396px;" data-options="editable:false">
							<option value="0">disabled</option>
						    <option value="1">enable</option>
						</select>
						<span id="m_ARP_PCICheckSpan" class="prompt" ></span>
					</div>
					<div class="epcConfigTrafficDiv">
						<label for="mARP_PVI">ARP PVI</label>
						<select id="m_ARP_PVI" class="easyui-combobox border border-box" style="height:27px;width:396px;" data-options="editable:false">
						    <option value="0">disabled</option>
						    <option value="1">enable</option>
						</select>
						<span id="m_ARP_PVICheckSpan" class="prompt" ></span>
					</div>
					<div class="epcConfigTrafficDiv">
						<label for="mPRECEDENCE">PRECEDENCE</label>
						<input id="m_PRECEDENCE" type="text" value="" maxlength="3" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" style="width:300px;" onblur="checkIntRange(this.id,this.value,0,255)" />
						<span id="m_PRECEDENCECheckSpan" class="prompt" ></span>
					</div>
					<div class="epcConfigTrafficDiv">
					    <label for="mbrUL">MBR UL (bps:1000~104857600)</label>
					    <input id="m_mbrUL" type="text" value="" maxlength="9" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" style="width:300px;" onblur="checkIntRange(this.id,this.value,1000,104857600)" />
					    <span id="m_mbrULCheckSpan" class="prompt" ></span>
					</div>
					<div class="epcConfigTrafficDiv">
					    <label for="mbrDL">MBR DL (bps:1000~104857600)</label>
					    <input id="m_mbrDL" type="text" value="" maxlength="9" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" style="width:300px;" onblur="checkIntRange(this.id,this.value,1000,104857600)" />
					    <span id="m_mbrDLCheckSpan" class="prompt" ></span>
					</div>
					<div class="epcConfigTrafficDiv">
					    <label for="gbrUL">GBR UL (bps:1000~104857600)</label>
					    <input id="m_gbrUL"  type="text" value="" maxlength="9" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" style="width:300px;"/>
					    <span id="m_gbrULCheckSpan" class="prompt" ></span>
					</div>
					<div class="epcConfigTrafficDiv">
					    <label for="gbrDL">GBR DL (bps:1000~104857600)</label>
					    <input id="m_gbrDL"  type="text" value="" maxlength="9" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" style="width:300px;"/>
					    <span id="m_gbrDLCheckSpan" class="prompt" ></span>
					</div>
					
					<!-- 表格  -- 选择PCC Rule -->
					<div>
						<%-- <div class="singleTitle">PCC Rule <%=rb.getString("XuanZe")%></div> --%>
						<ul class="omcPageTitleContainer" style="padding-left:0px;">
							<li class="default">PCC Rule<%=rb.getString("XuanZe")%></li>
						</ul>
					  	<div class="modifyContainer">
					      	<div class="leftTableDiv">
						      	<!-- 工具栏 修改PCC Rule -->
								<div class="queryGroup" style="margin-top:10px;margin-bottom:15px;">
						  			<p><%=rb.getString("QuanBuLieBiao")%></p>
						  			<input id="query_pcrf_pf_id_m" placeholder="PF ID" style="width:200px;"/>
									<b class="el-icon el-icon-common-search" onclick="serialPCRFPF_ID_m()"></b>
						  		</div>
										
								<div style="height:400px;">
									<table class="easyui-datagrid" id="m_pcc_rule_table_list" fit="true" data-options="fitColumns:true,singleSelect:false,border:true,
						                    rownumbers:true,url:'${ctx}/epc/configuration/getFilterInfos.action?EPC_ID=-1',pageSize:${pageSize},pageList:${pageList},striped:true,
						                    pagination:true,pagePosition:'bottom',onBeforeLoad:onBeforeladtaskTable,
						                    onLoadError:datagridLoadError">
										<thead>
											<tr>
												<th data-options="field:'PF_ID'" width="80">PF ID</th>
												<th data-options="field:'APP_NAME'" width="150"><%=rb.getString("APPMingCheng")%></th>
											</tr>
										</thead>
								    </table> 
								</div>   	
					     	</div>
					     	
					      	<div class="centerBtnGroup" style="width: 80px;">
					      		<a href="javascript:void(0);" class="arrow_right" style="margin-top:110px" onclick="addSelectedCell_auto_confirm('m_pcc_rule_table_list','selectedmRule_auto_confirm')"></a>
								<a href="javascript:void(0);" class="arrow_left" style="margin-top: 10px;" onclick="delSelectedCell_auto_confirm('selectedmRule_auto_confirm')"></a>
					      	</div>
					      	
					      	<div class="rightTableDiv">
					      	 	<p style="height:72px;line-height:35px;"><%=rb.getString("YiXuanLieBiao")%></p>
					     	 	<div style="height:400px;">
						      	 	<table class="easyui-datagrid" id="selectedmRule_auto_confirm" fit="true" data-options="fitColumns:true,singleSelect:false,border:true,
						                    rownumbers:true,url:'',pageSize:${pageSize},pageList:${pageList},striped:true,data:[],
						                    pagination:true,pagePosition:'bottom',toolbar:'',onLoadSuccess:loadSuccess,
						                    onLoadError:datagridLoadError">
										<thead>
											<tr>
												<th data-options="field:'PF_ID'" width="80">PF ID</th>
												<%--<th data-options="field:'APP_NAME'" width="150"><%=rb.getString("APPMingCheng")%></th>--%>
											</tr>
										</thead>
								    </table>
								</div>
					      	</div>
					 	</div>
					</div>
					<div class="submitConfig" style="margin-top:20px;margin-left:0;margin-bottom:40px">
				    	<a href="#" class="linkbutton" style="float:left;" onclick="updateTrafficConfiguration()"><span><%=rb.getString("BaoCun")%></span></a>
				  	</div>
				</div>
			</div>			
		</div>
		
		<!-- IMSI Configuration -->
		<div class="EPCRightItem">
			<div class="singleTitle">IMSI Configuration</div>
			<!-- 右上角添加按钮 -->
			<div class="circleIcon" id="addeGWIMSITRAFFIC" myOpType="1" onclick="operEpcConfiguration('ON','addImsiTrafficSetting')">
				<span class="el-icon el-icon-circle-add"></span>
				<div class="titleButtonText"><%=rb.getString("TianJia")%></div>
			</div>
			
			<!-- 工具栏  --  查询 -->
		  	<div id="queryImsiTrafficStting" class="toolbarContainer" style="margin-left:15px;">	
		  		<div class="queryGroup">
		  			<input id="sprImsiTraffic" placeholder="IMSI"/>
		  			<b class="el-icon el-icon-common-search" onclick="queryImsiTraffic()"></b>
		  		</div>
			</div>
			
			<!-- 表格 -->
			<div class="contentDiv">
				 <table class="easyui-datagrid" id="epc_spr_imsi_traffic_table" fit="true" data-options="fitColumns:true,singleSelect:true,border:false,toolbar:'#queryImsiTrafficStting',
		                    rownumbers:true,url:'${ctx}/epc/configuration/getImsiTrafficInfosPage.action?EPC_ID=-1',pageSize:${pageSize},pageList:${pageList},striped:true,
		                    pagination:true,pagePosition:'bottom',onLoadSuccess: datagridLoadSuccess,
		                    onLoadError:datagridLoadError">
						<thead>
							<tr>
							    <th data-options="field:'IMSI',sortable:true" width="10">IMSI</th>
								<th data-options="field:'APN_NAME',sortable:true" width="10"><%=rb.getString("APNMingCheng")%></th>
								<th data-options="field:'PCC_LIST',sortable:true" width="30">PCC LIST</th>
								<th data-options="field:'operation',formatter: operFormatterImsiTraffic,fixed:true" width="80"><%=rb.getString("CaoZuo")%></th>
								<th data-options="field:'EPC_SERVER_IP'" hidden="true" width="1"></th>
								<th data-options="field:'EPC_PORT'" hidden="true" width="1"></th>
								<th data-options="field:'EPC_ID'" hidden="true" width="1"></th>
							</tr>
						</thead>
			     </table>
			</div>
			
			<!-- 添加 imsi -->
			<div class="addImsiTrafficSetting" style="left:0;right:0;overflow:auto">
				<div class="shuntChoose"></div>
				<div class="epcConfigInfoItemDiv">
					<label for="IMSI">IMSI</label>
				   	<input id="IMSI" type="text" value="" maxlength="15" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" style="width:300px;" onblur="checkLongRange(this.id,this.value,15)" />
				  	<span id="IMSICheckSpan" class="prompt" ></span>
				</div>
				<div class="epcConfigInfoItemDiv">
					<label for="APN_NAME"><%=rb.getString("APNMingCheng")%></label>
				  	<input id="APN_NAME" type="text" value="" maxlength="63" class="easyui-validatebox border border-box" style="width:300px;" onblur="if(checkRangLength(this.id,this.value,1,100)){checkName(this.id,this.value);}"/>
				  	<span id="APN_NAMECheckSpan" class="prompt" ></span>
				</div>
				
				<!-- 下拉框中 表格-->
				<ul class="omcPageTitleContainer" style="padding-left:0px;">
					<li class="default">IMSI<%=rb.getString("XuanZe")%></li>
				</ul>
				<div class="modifyContainer">
					<div class="leftTableDiv">
				      	<!-- 工具栏 查询PCC Name -->
				  		<div style="margin-top:10px;margin-bottom:15px;">
				  			<p><%=rb.getString("QuanBuLieBiao")%></p>
				  			<div class='queryGroup' style='margin-left:0px;'>
				  				<input id="query_imsi_pcc_name" placeholder="<%=rb.getString("PCCMingCheng")%>" style="width:200px;"/>
								<b class="el-icon el-icon-common-search" onclick="serialPccName()"></b>
				  			</div>
				  		</div>
				  		
						<div style="height:400px;">
							<table class="easyui-datagrid" id="imsi_traffic_table_list" fit="true" data-options="fitColumns:true,singleSelect:false,border:true,
				                    rownumbers:true,url:'${ctx}/epc/configuration/getTrafficInfos.action?EPC_ID=-1',pageSize:${pageSize},pageList:${pageList},striped:true,
				                    pagination:true,pagePosition:'bottom',onBeforeLoad:onBeforeladtaskTable,
				                    onLoadError:datagridLoadError">
								<thead>
									<tr>
										<!-- <th data-options="field:'EPC_NAME'" width="150">EPC_NAME</th> -->
										<th data-options="field:'PCC_NAME'" width="200"><%=rb.getString("PCCMingCheng")%></th>
									</tr>
								</thead>
						    </table> 
						</div> 	
					</div>
				    <div class="centerBtnGroup" style="width: 80px;">
						<a href="javascript:void(0 );" class="arrow_right" style="margin-top:110px" onclick="addSelectedCell_auto_confirm('imsi_traffic_table_list','selectedImsi_auto_confirm')"></a>
						<a href="javascript:void(0);" class="arrow_left" style="margin-top: 10px;" onclick="delSelectedCell_auto_confirm('selectedImsi_auto_confirm')"></a>
				    </div>
				    <div class="rightTableDiv">
				    	<p style="height:72px;line-height:35px;"><%=rb.getString("YiXuanLieBiao")%></p>
				     	<div style="height:400px;">
					    	<table class="easyui-datagrid" id="selectedImsi_auto_confirm" fit="true" data-options="fitColumns:true,singleSelect:false,border:true,
				                    rownumbers:true,url:'',pageSize:${pageSize},pageList:${pageList},striped:true,data:[],
				                    pagination:true,pagePosition:'bottom',toolbar:'',onLoadSuccess:loadSuccess,
				                    onLoadError:datagridLoadError">
								<thead>
									<tr>
										<th data-options="field:'PCC_NAME'" width="200"><%=rb.getString("PCCMingCheng")%></th>
									</tr>
								</thead>
						    </table>
						</div>
				    </div>
				</div>
				
				<div style="margin-top:20px;margin-left:20px;height:30px;margin-bottom:20px">
			      	<a href="#" class="linkbutton" style="float:left;" onclick="addImsiTrafficConfiguration()"><span><%=rb.getString("BaoCun")%></span></a>
			  	</div>
			</div>
			
			<!-- 修改 IMSI -->
			<div class="modifyImsiTrafficSetting" style="left:0px;right:0px;top:41px;bottom:20px;padding-bottom:20px;overflow:auto">
				<div class="newEGWTit">				
			    	<div id="trafficCpeName" style="display:none;color:#7993B6;font-size:16px;"></div>
			    </div>
			    <div class="epcConfigInfoItem">
					<div class="epcConfigTrafficDiv">
					    <label for="MIMSI">IMSI</label>
					    <input id="m_IMSI"  type="text" value="" maxlength="15" disabled="disabled" onkeyup="value=value.replace(/[^\d]/g,'')" class="easyui-validatebox border border-box" style="width:300px;" onblur="checkIntRange(this.id,this.value,15,15)" />
					    <span id="m_IMSICheckSpan" class="prompt" ></span>
					</div>
					<div class="epcConfigTrafficDiv">
					    <label for="APN_NAME"><%=rb.getString("APNMingCheng")%></label>
					    <input id="m_APN_NAME" type="text" value="" maxlength="100" disabled="disabled" class="easyui-validatebox border border-box" style="width:300px;" onblur="if(checkRangLength(this.id,this.value,1,100)){checkName(this.id,this.value);}" />
					    <span id="m_APN_NAMECheckSpan" class="prompt" ></span>
					</div>
					  
					<ul class="omcPageTitleContainer" style="padding-left:0px;">
						<li class="default">IMSI<%=rb.getString("XuanZe")%></li>
					</ul>
					<div class="modifyContainer">
						<div class="leftTableDiv">
					      	<!-- 工具栏 修改IMSI -->
					  		<div class="queryGroup" style="margin-top:10px;margin-bottom:15px;">
					  			<p><%=rb.getString("QuanBuLieBiao")%></p>
					  			<input id="query_imsi_pcc_name_m" placeholder="<%=rb.getString("PCCMingCheng")%>" style="width:200px;"/>
								<b class="el-icon el-icon-common-search" onclick="serialPccName_m()"></b>
					  		</div>
								
							<div style="height:400px;">
								<table class="easyui-datagrid" id="m_imsi_traffic_table_list" fit="true" data-options="fitColumns:true,singleSelect:false,border:true,
					                    rownumbers:true,url:'${ctx}/epc/configuration/getTrafficInfos.action',pageSize:${pageSize},pageList:${pageList},striped:true,
					                    pagination:true,pagePosition:'bottom',onBeforeLoad:onBeforeladtaskTable,
					                    onLoadError:datagridLoadError">
									<thead>
										<tr>
										   <!-- <th data-options="field:'EPC_NAME'" width="150">EPC_NAME</th>  -->
											<th data-options="field:'PCC_NAME'" width="200"><%=rb.getString("PCCMingCheng")%></th>
										</tr>
									</thead>
							    </table>
							</div>  	
					    </div>
					    <div class="centerBtnGroup" style="width: 80px;">
					    	<a href="javascript:void(0);" class="arrow_right" style="margin-top:110px" onclick="addSelectedCell_auto_confirm('m_imsi_traffic_table_list','selectedmimsi_auto_confirm')"></a>
							<a href="javascript:void(0);" class="arrow_left" style="margin-top: 10px;" onclick="delSelectedCell_auto_confirm('selectedmimsi_auto_confirm')"></a>
					    </div>
					    <div class="rightTableDiv">
					      	<p style="height:72px;line-height:35px;"><%=rb.getString("YiXuanLieBiao")%></p>
					     	<div style="height:400px;">
								<table class="easyui-datagrid" id="selectedmimsi_auto_confirm" fit="true" data-options="fitColumns:true,singleSelect:false,border:true,
					                    rownumbers:true,url:'',pageSize:${pageSize},pageList:${pageList},striped:true,data:[],
					                    pagination:true,pagePosition:'bottom',toolbar:'',onLoadSuccess:loadSuccess,
					                    onLoadError:datagridLoadError">
									<thead>
										<tr>
											<th data-options="field:'PCC_NAME'" width="200"><%=rb.getString("PCCMingCheng")%></th>
										</tr>
									</thead>
							    </table>
							</div>
					    </div>
					</div>
					<div class="submitConfig" style="margin-top:20px;margin-left:0px;">
				    	<a href="#" class="linkbutton" style="float:left;" onclick="updateImsiTrafficConfiguration()"><span><%=rb.getString("BaoCun")%></span></a>
				  	</div>
			  	</div>
			</div>
		</div>
		
		<!-- APN Configuration -->
		<div class="EPCRightItem">
			<div class="singleTitle">APN Configuration</div>
			<!-- 右上角添加按钮 -->
			<div class="circleIcon" id="addeGWAPNTRAFFIC" myOpType="1" onclick="operEpcConfiguration('ON','addApnTrafficSetting')">
				<span class="el-icon el-icon-circle-add"></span>
				<div class="titleButtonText"><%=rb.getString("TianJia")%></div>
			</div>
			
			<!-- 搜索框 -->
			<div id="queryApnTrafficStting" class="toolbarContainer" style="margin-left:15px;">
		  		<div class="queryGroup">
		  			<input id="sprApnTraffic" placeholder="<%=rb.getString("APNMingCheng")%>"/>
					<b class="el-icon el-icon-common-search" onclick="queryApnTraffic()"></b>
		  		</div>
			</div>
			<!-- 表格 -->
			<div class="contentDiv">
				<table class="easyui-datagrid" id="epc_spr_apn_traffic_table" fit="true" data-options="fitColumns:true,singleSelect:true,border:false,toolbar:'#queryApnTrafficStting',
	                    rownumbers:true,url:'${ctx}/epc/configuration/getApnTrafficInfosPage.action?EPC_ID=-1',pageSize:${pageSize},pageList:${pageList},striped:true,
	                    pagination:true,pagePosition:'bottom',onLoadSuccess: datagridLoadSuccess,
	                    onLoadError:datagridLoadError">
					<thead>
						<tr>
						    <th data-options="field:'APN_NAME',sortable:true" width="10"><%=rb.getString("APNMingCheng")%></th>
							<th data-options="field:'PCC_LIST',sortable:true" width="20">PCC_LIST</th>
							<th data-options="field:'operation',formatter: operFormatterApnTraffic,fixed:true" width="80"><%=rb.getString("CaoZuo")%></th>
							<th data-options="field:'EPC_SERVER_IP'" hidden="true" width="1"></th>
							<th data-options="field:'EPC_PORT'" hidden="true" width="1"></th>
							<th data-options="field:'EPC_ID'" hidden="true" width="1"></th>
						</tr>
					</thead>
			 	</table>
			</div>
			
			<!-- 添加APN Configuration -->
			<div class="addApnTrafficSetting" style="left:0;right:0;overflow-y:auto">
				<div class="shuntChoose">
				</div>
				<div class="epcConfigInfoItemDiv">
				  	<label for="qci"><%=rb.getString("APNMingCheng")%></label>
				  	<input id="apn_apn_name" type="text" value="" maxlength="100" class="easyui-validatebox border border-box" style="width:300px;" onblur="if(checkRangLength(this.id,this.value,1,100)){checkName(this.id,this.value);}" />
				  	<span id="apn_apn_nameCheckSpan" class="prompt" ></span>
				</div>
				
				<!-- 下拉框中 表格-->
				<ul class="omcPageTitleContainer" style="padding-left:0px;">
					<li class="default">APN<%=rb.getString("XuanZe")%></li>
				</ul>
				<div class="modifyContainer">
					<div class="leftTableDiv">
				    <!-- 工具栏 修改PCC Rule -->
				  		<div style="margin-top:10px;margin-bottom:15px;">
				  			<p><%=rb.getString("QuanBuLieBiao")%></p>
				  			<div class='queryGroup' style='margin-left:0px;'>
				  				<input id="query_apn_pcc_name" placeholder="<%=rb.getString("PCCMingCheng")%>" style="width:200px;"/>
								<b class="el-icon el-icon-common-search" onclick="serialApnPccName()"></b>
				  			</div>
				  		</div>

						<div style="height:400px;">
							<table class="easyui-datagrid" id="apn_traffic_table_list" fit="true" data-options="fitColumns:true,singleSelect:false,border:true,
				                    rownumbers:true,url:'${ctx}/epc/configuration/getTrafficInfos.action',pageSize:${pageSize},pageList:${pageList},striped:true,
				                    pagination:true,pagePosition:'bottom',onBeforeLoad:onBeforeladtaskTable,
				                    onLoadError:datagridLoadError">
								<thead>
									<tr>
										<!-- <th data-options="field:'EPC_NAME'" width="150">EPC_NAME</th>  -->
										<th data-options="field:'PCC_NAME'" width="200"><%=rb.getString("PCCMingCheng")%></th>
									</tr>
								</thead>
						    </table> 
						</div>  	
					</div>
				    <div class="centerBtnGroup" style="width: 80px;">
						<a href="javascript:void(0);" class="arrow_right" style="margin-top:110px" onclick="addSelectedCell_auto_confirm('apn_traffic_table_list','selectedApn_auto_confirm')"></a>
						<a href="javascript:void(0);" class="arrow_left" style="margin-top: 10px;" onclick="delSelectedCell_auto_confirm('selectedApn_auto_confirm')"></a>
				    </div>
				    <div class="rightTableDiv">
				    	<p style="height:72px;line-height:35px;"><%=rb.getString("YiXuanLieBiao")%></p>
				     	<div style="height:400px;">
					    	<table class="easyui-datagrid" id="selectedApn_auto_confirm" fit="true" data-options="fitColumns:true,singleSelect:false,border:true,
				                    rownumbers:true,url:'',pageSize:${pageSize},pageList:${pageList},striped:true,data:[],
				                    pagination:true,pagePosition:'bottom',toolbar:'',onLoadSuccess:loadSuccess,
				                    onLoadError:datagridLoadError">
								<thead>
									<tr>
										<th data-options="field:'PCC_NAME'" width="200"><%=rb.getString("PCCMingCheng")%></th>
									</tr>
								</thead>
							</table>
						</div>
					</div>
				</div>
			
				<div style="margin-top:20px;margin-left:20px;margin-bottom:20px;height:30px;">
			      	<a href="#" class="linkbutton" style="float:left;" onclick="addApnTrafficConfiguration()"><span><%=rb.getString("BaoCun")%></span></a>
			  	</div>
			</div>
			
			<!-- 编辑 apn -->
			<div class="modifyApnTrafficSetting">
				<div class="newEGWTit">				
			    	<div id="trafficCpeName" style="display:none;color:#7993B6;font-size:16px;"></div>
			    </div>
			    <div class="epcConfigInfoItem">
					<div class="epcConfigTrafficDiv">
				   		<label for="APN_NAME"><%=rb.getString("APNMingCheng")%></label>
				    	<input id="mm_APN_NAME" type="text" value="" maxlength="100" disabled="disabled" class="easyui-validatebox border border-box" style="width:300px;" onblur="if(checkRangLength(this.id,this.value,1,100)){checkName(this.id,this.value);}" />
				    	<span id="mm_APN_NAMECheckSpan" class="prompt" ></span>
				  	</div>
				  
				  	<ul class="omcPageTitleContainer" style="padding-left:0px;">
						<li class="default">APN<%=rb.getString("XuanZe")%></li>
					</ul>
					<div class="modifyContainer">
						<div class="leftTableDiv">
					    	<!-- 工具栏 修改APN -->
					  		<div class="queryGroup" style="margin-top:10px;margin-bottom:15px;">
					  			<p><%=rb.getString("QuanBuLieBiao")%></p>
					  			<input id="query_apn_pcc_name_m" placeholder="<%=rb.getString("PCCMingCheng")%>" style="width:200px;"/>
								<b class="el-icon el-icon-common-search" onclick="serialApnPccName_m()"></b>
					  		</div>
								
							<div style="height:400px;">
								<table class="easyui-datagrid" id="mapn_traffic_table_list" fit="true" data-options="fitColumns:true,singleSelect:false,border:true,
					                    rownumbers:true,url:'${ctx}/epc/configuration/getTrafficInfos.action',pageSize:${pageSize},pageList:${pageList},striped:true,
					                    pagination:true,pagePosition:'bottom',onBeforeLoad:onBeforeladtaskTable,
					                    onLoadError:datagridLoadError">
									<thead>
										<tr>
										    <!-- <th data-options="field:'EPC_NAME'" width="150">EPC_NAME</th>  -->
											<th data-options="field:'PCC_NAME'" width="200"><%=rb.getString("PCCMingCheng")%></th>
										</tr>
									</thead>
							    </table>
							</div>  	
						</div>
					    <div class="centerBtnGroup" style="width: 80px;">
					    	<a href="javascript:void(0);" class="arrow_right" style="margin-top:110px;" onclick="addSelectedCell_auto_confirm('mapn_traffic_table_list','selectedmApn_auto_confirm')"></a>
							<a href="javascript:void(0);" class="arrow_left" style="margin-top: 10px;" onclick="delSelectedCell_auto_confirm('selectedmApn_auto_confirm')"></a>
					    </div>
					    <div class="rightTableDiv">
					        <p style="height:72px;line-height:35px;"><%=rb.getString("YiXuanLieBiao")%></p>
					     	<div style="height:400px;">
								<table class="easyui-datagrid" id="selectedmApn_auto_confirm" fit="true" data-options="fitColumns:true,singleSelect:false,border:true,
					                    rownumbers:true,url:'',pageSize:${pageSize},pageList:${pageList},striped:true,data:[],
					                    pagination:true,pagePosition:'bottom',toolbar:'',onLoadSuccess:loadSuccess,
					                    onLoadError:datagridLoadError">
									<thead>
										<tr>
											<th data-options="field:'PCC_NAME'" width="200"><%=rb.getString("PCCMingCheng")%></th>
										</tr>
									</thead>
							    </table>
							</div>
				      	</div>
				  	</div>
				 	<div class="submitConfig" style="margin-top:20px;margin-left:0px;">
			    	  	<a href="#" class="linkbutton" style="float:left;" onclick="updateApnTrafficConfiguration()"><span><%=rb.getString("BaoCun")%></span></a>
			  	 	</div>
			  	</div>
			</div>
		</div>
	</div>
</div>
<script>
	 var selectedEPCItem = 0;
	 $(function(){
		 var epcServer='${epcServer}';
		 var selectEPCData = $.parseJSON(epcServer);
		 //选择EPC
		/*  var newDataSelect = [{
			 ADDR_TYPE: "1",
			 CREATE_TIME:1516410452000,
		     EPC_ID: 2,
			 IP:"192.168.5.4",
			 NAME:"wangjiwei",
			 OPERATOR:"admin",
			 OPER_TIME:1516410452000,
			 PORT:"90",
			 },
			 {
				 ADDR_TYPE: "2",
				 CREATE_TIME:1516410452000,
			     EPC_ID:3,
				 IP:"192.168.5.9",
				 NAME:"pengfei",
				 OPERATOR:"admin",
				 OPER_TIME:1516410452000,
				 PORT:"70",
				 },
			 ] */
		 $("#PCCchoseEPC").combobox({
		    	data:selectEPCData,
		    	panelHeight:50,
		        valueField: 'EPC_ID',
		        textField: 'NAME',
		        onSelect:chooseShunt,
		        onLoadSuccess:function(){
		        	$("#PCCchoseEPC").combobox("panel").closest(".combo-p").addClass("zIndex");
		        }
		        
		 })
		
		 //var adddata =  $("#PCCchoseEPC").combobox('getValue');
		 //新功能
		 var load_pf=0,load_rule=0,load_imsi=0,load_apn=0;
		 $("#PCRFList li").click(function(){
			 var _thisindex = $(this).index();
			 $(this).siblings().removeClass("pitchOnItem");
			 $(this).addClass("pitchOnItem");
			 $(".EPCRightItem").eq(_thisindex).css("display","block");
			 $(".EPCRightItem").eq(_thisindex).siblings().css("display","none");	
		 	 var selectValue = $("#PCCchoseEPC").combobox("getValues");
		 	 if(selectValue[0] == ""){
		 		switch(_thisindex){
				 	case 0:
				 		$("#filter_table").datagrid("resize");
				 		$("#sprPacketFilter").val("");
				 		selectedEPCItem = 0;
				 	break;
				 	case 1:
				 		$("#traffic_table").datagrid("resize");
				 		$("#sprCommonTraffic").val("");
				 		selectedEPCItem = 1;
				 	break;
				 	case 2:
				 		$("#epc_spr_imsi_traffic_table").datagrid("resize");
				 		$("#sprImsiTraffic").val("");
				 		selectedEPCItem = 2;
				 	break;
				 	case 3:
				 		$("#epc_spr_apn_traffic_table").datagrid("resize");
				 		$("#sprApnTraffic").val("");
				 		selectedEPCItem = 3;
				 	break;
				 }
		 	 }else{
				 var paramsArray = $("#PCCchoseEPC").combobox("getData");
				 //选中的参数
				 var dataparams = paramsArray.filter(function(item){
				 		return item.EPC_ID == selectValue;
				 });
				 var epc_id = dataparams[0].EPC_ID;
				 var epc_ip = dataparams[0].IP;
				 var epc_port = dataparams[0].PORT;
				 var param = {};
				 param["EPC_ID"]=dataparams[0].EPC_ID;
				 param["EPC_IP"]=dataparams[0].IP;
				 param["EPC_PORT"]=dataparams[0].PORT;
				 //每次切换需要重置
			  	 //解除click事件
			  	$("#addeGWFilter").removeAttr("onclick");
			  	$("#addeGWFilter").attr("onclick","operEpcConfiguration('ON','addFilterSetting')");
			  	$("#addeGWFilter .titleButtonText").text("<%=rb.getString("TianJia")%>");
				$("#addeGWFilter .circleBg").addClass("add_circle");
				$("#addeGWFilter .circleBg").removeClass("close_circle");
				$(".modifyFilterSetting").slideUp(300);
				$(".addFilterSetting").slideUp(300);
				
			  	$("#addeGWTraffic").removeAttr("onclick");
			  	$("#addeGWTraffic").attr("onclick","operEpcConfiguration('ON','addTrafficSetting')");
			  	$("#addeGWTraffic .titleButtonText").text("<%=rb.getString("TianJia")%>");
				$("#addeGWTraffic .circleBg").addClass("add_circle");
				$("#addeGWTraffic .circleBg").removeClass("close_circle");
				$(".modifyTrafficSetting").slideUp(300);
				$(".addTrafficSetting").slideUp(300);
				
			  	$("#addeGWIMSITRAFFIC").removeAttr("onclick");
			  	$("#addeGWIMSITRAFFIC").attr("onclick","operEpcConfiguration('OFF','addImsiTrafficSetting');");
				$("#addeGWIMSITRAFFIC .titleButtonText").text("<%=rb.getString("TianJia")%>");
				$("#addeGWIMSITRAFFIC .circleBg").addClass("add_circle");
				$("#addeGWIMSITRAFFIC .circleBg").removeClass("close_circle");
				$(".modifyImsiTrafficSetting").slideUp(300); 
				$(".addImsiTrafficSetting").slideUp(300); 
				
			  	$("#addeGWAPNTRAFFIC").removeAttr("onclick");
			  	$("#addeGWAPNTRAFFIC").attr("onclick","operEpcConfiguration('OFF','addApnTrafficSetting')");
			  	$("#addeGWAPNTRAFFIC .titleButtonText").text("<%=rb.getString("TianJia")%>");
				$("#addeGWAPNTRAFFIC .circleBg").addClass("add_circle");
				$("#addeGWAPNTRAFFIC .circleBg").removeClass("close_circle");
				$(".modifyApnTrafficSetting").slideUp(300);
				$(".addApnTrafficSetting").slideUp(300);
				showPCRFAddFlag = true;
				 switch(_thisindex){
				 	case 0:
				 		$("#filter_table").datagrid("resize");
				 		$("#sprPacketFilter").val("");
				 		if( epc_id!=undefined && epc_id!="" && load_pf==0 ){
						 	$("#select_epc_pf_val").text(name);
							savingCover();
							$.post("${ctx}/epc/configuration/createFilterInfos.action", param, function(data){
								 cancelSavingCover();
								 doSearchUrl('filter_table', param,'${ctx}/epc/configuration/getFilterInfos.action');
							});
							load_pf=1;
					 	}else if( epc_id!=undefined && epc_id!="" && load_pf!=0){
							 doSearchUrl('filter_table', param,'${ctx}/epc/configuration/getFilterInfos.action');
					 	}
				 		selectedEPCItem = 0;
				 	break;
				 	case 1:
				 		$("#traffic_table").datagrid("resize");
				 		$("#sprCommonTraffic").val("");
					 	if( epc_id!=undefined && epc_id!="" && load_rule==0 ){
						 	$("#select_epc_rule_val").text(name);
							savingCover();
							$.post("${ctx}/epc/configuration/createTrafficInfos.action", param, function(data){
								 cancelSavingCover();
								 doSearchUrl('traffic_table', param,'${ctx}/epc/configuration/getTrafficInfos.action');
							});
							load_rule=1;
					 	}else if( epc_id!=undefined && epc_id!="" && load_rule!=0){
							doSearchUrl('traffic_table', param,'${ctx}/epc/configuration/getTrafficInfos.action');
					 	}
				 		selectedEPCItem = 1;
				 	break;
				 	case 2:
				 		$("#epc_spr_imsi_traffic_table").datagrid("resize");
				 		$("#sprImsiTraffic").val("");
					 	if( epc_id!=undefined && epc_id!="" && load_imsi==0 ){
						 	$("#select_epc_imsi_val").text(name);
							savingCover();
							 $.post("${ctx}/epc/configuration/createImsiTraffic.action", param, function(data){
								 cancelSavingCover();
						  		 doSearchUrl('epc_spr_imsi_traffic_table', param,'${ctx}/epc/configuration/getImsiTrafficInfosPage.action');
							 }, "json");
							 load_imsi=1;
					 	}else if( epc_id!=undefined && epc_id!="" && load_imsi!=0){
					  		 doSearchUrl('epc_spr_imsi_traffic_table', param,'${ctx}/epc/configuration/getImsiTrafficInfosPage.action');
					 	}
				 		selectedEPCItem = 2;
				 	break;
				 	case 3:
				 		$("#epc_spr_apn_traffic_table").datagrid("resize");
				 		$("#sprApnTraffic").val("");
					 	if( epc_id!=undefined && epc_id!="" && load_apn==0 ){
						 	$("#select_epc_apn_val").text(name);
							savingCover();
							 $.post("${ctx}/epc/configuration/createApnTraffic.action", param, function(data){
								 cancelSavingCover();
								 doSearchUrl('epc_spr_apn_traffic_table', param,'${ctx}/epc/configuration/getApnTrafficInfosPage.action');
							 });
							 load_apn=1;
					 	}else if( epc_id!=undefined && epc_id!="" && load_apn!=0 ){
							 doSearchUrl('epc_spr_apn_traffic_table', param,'${ctx}/epc/configuration/getApnTrafficInfosPage.action');
					 	}
				 		selectedEPCItem = 3;
				 	break;
				 }
		 	 }
		 })
		 //新功能结束
		
		 
		 
		 
		 //PF回车查询
		 $("#sprPacketFilter").keyup(function(event){
				if(event.keyCode==13){
					queryPacketFilter();
				}
		 });
		 //RULE回车查询
		 $("#sprCommonTraffic").keyup(function(event){
				if(event.keyCode==13){
					queryPCCRULEFilter();
				}
		 });
		 //imsi回车查询
		 $("#sprImsiTraffic").keyup(function(event){
				if(event.keyCode==13){
					queryImsiTraffic();
				}
	     });
		 //apn回车查询
		 $("#sprApnTraffic").keyup(function(event){
				if(event.keyCode==13){
					queryApnTraffic();
				}
		 });
		 
		$("#uploadForm_deviceGroupCell input[name='uploadFile']").bind("change", function() {
			$("#ImportDeviceGroupCellFile input[name='uploadFilePath']").val(this.value);
			if (this.value) {
				$("#winUploadDeviceGroupCellFile input[name='uploadFilePath']").removeClass("err_border");
			}
		});
		$("#protocol").combobox({
			onSelect:function(){
				//切换PROTOCOL控制IP_MASK是否必选
				var pcl=$("#protocol").combobox("getValue");
				if(pcl != "ip"){
					$(".filterIpMaskIsNullImg").hide();
				}else{
					$(".filterIpMaskIsNullImg").show();
				}
			}
		});
		
		$("#m_protocol").combobox({
			onSelect:function(){
				//切换PROTOCOL控制IP_MASK是否必选
				var pcl=$("#m_protocol").combobox("getValue");
				if(pcl != "ip"){
					$(".mfilterIpMaskIsNullImg").hide();
				}else{
					$(".mfilterIpMaskIsNullImg").show();
				}
			}
		});
		
		$("#filterIpMask").blur(function(){
			var pcl=$("#protocol").combobox("getValue");
			if(pcl == "ip"){
				checkIpMaskFormat(this.id,this.value);
			}
		})
		
		$("#m_filterIpMask").blur(function(){
			var pcl=$("#m_protocol").combobox("getValue");
			if(pcl == "ip"){
				checkIpMaskFormat(this.id,this.value);
			}
		});
	});

	//打开文件选择窗口
	function scanClick_deviceGroup() {
		$("#uploadForm_deviceGroupCell input[name='uploadFile']").click();
	}
	
	//上传文件
	function uploadFile_deviceGroupCell() {
		if ($("#uploadForm_deviceGroupCell input[name='uploadFile']")[0].files.length == 0) {
			$("#winUploadDeviceGroupCellFile input[name='uploadFilePath']").addClass("err_border").fadeOut().fadeIn();
			return;
		}
		$("#uploadForm_deviceGroupCell").form("submit", {
			success: function (data) {
				
	        },
	        onSubmit: function(param){
				var bool = checkParams(param);
				if(!bool) return false;
			}
		});
	}
	
	function operFormatter(value, rowData, rowIndex){
		var res="";
			res="<div class='grid-edit-btn-div' style='margin-left: 15px' title='modify' onclick='modifyTrafficConfiguration(\""+rowIndex+"\")'></div>";
			res+="<div class='grid-del-btn-div' style='margin-left: 15px' title='delete' onclick='deleteTrafficSetting(\""+rowIndex+"\")'></div>";
		return res;
	}
	
  	function operFormatterFilter(value, rowData, rowIndex){
		var res="";//epc要求暂时屏蔽 后续他们程序支撑修改删除后 可以放开
			res="<div class='grid-edit-btn-div' style='margin-left: 15px' title='modify' onclick='modifyFilterConfiguration(\""+rowIndex+"\")'></div>";
			res+="<div class='grid-del-btn-div' style='margin-left: 15px' title='delete' onclick='deleteFilterConfiguration(\""+rowIndex+"\")'></div>";
		return res;
  	}
  	
  	function operFormatterImsiTraffic(value, rowData, rowIndex){
		var res="";
		res="<div class='grid-edit-btn-div' style='margin-left: 15px' title='modify' onclick='modifyImsiTrafficConfiguration(\""+rowIndex+"\")'></div>";
		res+="<div class='grid-del-btn-div' style='margin-left: 15px' title='delete' onclick='deleteImsiTrafficSetting(\""+rowIndex+"\")'></div>";
		return res;
 	 }
  	
  	function operFormatterApnTraffic(value, rowData, rowIndex){
		var res="";
			res="<div class='grid-edit-btn-div' style='margin-left: 15px' title='modify' onclick='modifyApnTrafficConfiguration(\""+rowIndex+"\")'></div>";
			res+="<div class='grid-del-btn-div' style='margin-left: 15px' title='delete' onclick='deleteApnTrafficSetting(\""+rowIndex+"\")'></div>";
		return res;
 	 }
  
 	//统一添加下拉、关闭事件 根据class判断 tag:on-打开 off-关闭 dclass:操作目标div class名称   
 	var showPCRFAddFlag = true;
 	function operEpcConfiguration(tag,dclass){
	  	var hearderClass="";
	  	//判断是否选中了EPC
	  	var epcId =  $("#PCCchoseEPC").combobox('getValue');
	  	if((epcId==undefined||epcId=="") && tag=="ON"){
	  		$('.prompt_msg').html("<%=rb.getString("QingXuanZheEPCFuWu")%>");
	  		var width = $('.prompt_msg').width();
	  		$('.prompt_msg').css("left",'50%');
	  		var left = parseFloat($('.prompt_msg').css("left")) - width/2;
	  		$('.prompt_msg').css("left",left+'px');
	  		$('.prompt_msg').animate({top:'55px'},200,function(){
	  			setTimeout(function(){
	  				$('.prompt_msg').animate({top:'-40px'},function(){
	  					$("#PCCchoseEPC").combobox("showPanel");
	  				})
	  			},3000)
	  		})
		 	return;
	  	}
	  	if(dclass=="addFilterSetting"){
  		  	hearderClass="#addeGWFilter ";
  		  	getProtocolCombobox();
  	  	}
        if(dclass=="addTrafficSetting"){
      	  	hearderClass="#addeGWTraffic ";
  	  	}
        if(dclass=="addImsiTrafficSetting"){
      	  	hearderClass="#addeGWIMSITRAFFIC ";
        }
        if(dclass=="addApnTrafficSetting"){
      	  	hearderClass="#addeGWAPNTRAFFIC ";
        }
	  	if(showPCRFAddFlag){//添加-打开功能
	  		showPCRFAddFlag = false;
		  	//初始化界面元素数据
		  	$(window).resize();
		  	//$(hearderClass).children().addClass("addToeGW");
		  	$("."+dclass).slideDown(500);
		  	//解除click事件
		  	$(hearderClass).removeAttr("onclick");
		  	//重新绑定
		  	$(hearderClass).attr("onclick","operEpcConfiguration('OFF','"+dclass+"')");
		  	$(hearderClass+" .titleButtonText").text("<%=rb.getString("GuanBi")%>");
			$(hearderClass+" .circleBg").removeClass("add_circle");
			$(hearderClass+" .circleBg").addClass("close_circle");
			$(".epcConfigInfoItemDiv span.prompt").text("");
		  	//如果是打开trafic窗口,则需要动态刷新PF_LIST,依赖于FILTER配置PF_ID,用自定义属性标识是否需要动态查询PF_LIST 0-不查 1-查 
		  	//设置的参数
		  	var selectValue = $("#PCCchoseEPC").combobox("getValues");
			var paramsArray = $("#PCCchoseEPC").combobox("getData");
			//选中的参数
			var dataparams = paramsArray.filter(function(item){
					return item.EPC_ID == selectValue;
			});
		  	if(dclass=="addTrafficSetting"){
			  	$(".addTrafficSetting .shuntChooseItem li").attr("pfs","1");
			  	/* var param = {};
			  	param["EPC_ID"]=epc_id;
			  	param["EPC_IP"]=epc_ip;       //要删除的
			  	param["EPC_PORT"]=epc_port; */
				var param = {};
				param["EPC_ID"]=dataparams[0].EPC_ID;
				param["EPC_IP"]=dataparams[0].IP;
				param["EPC_PORT"]=dataparams[0].PORT;
			  	$.post("${ctx}/epc/configuration/createFilterInfos.action", param, function(data){
					 doSearchUrl('pcc_rule_table_list', param,'${ctx}/epc/configuration/getFilterInfos.action');
			  	});
			  	$("#selectedRule_auto_confirm").datagrid('loadData',{total:0,rows:[]});
		  	}
		  	if(dclass=="addImsiTrafficSetting"){
			  	var param = {};
				param["EPC_ID"]=dataparams[0].EPC_ID;
				param["EPC_IP"]=dataparams[0].IP;
				param["EPC_PORT"]=dataparams[0].PORT;
			  	$.post("${ctx}/epc/configuration/createTrafficInfos.action", param, function(data){
					 doSearchUrl('imsi_traffic_table_list', param,'${ctx}/epc/configuration/getTrafficInfos.action');
			  	});
			  	$("#selectedImsi_auto_confirm").datagrid('loadData',{total:0,rows:[]});
			  	$("#IMSI,#APN_NAME").val("");
		  	}
		  	if(dclass=="addApnTrafficSetting"){
			  	/* var param = {};
			  	param["EPC_ID"]=epc_id;
			  	param["EPC_IP"]=epc_ip;
			  	param["EPC_PORT"]=epc_port; */
			  	var param = {};
				param["EPC_ID"]=dataparams[0].EPC_ID;
				param["EPC_IP"]=dataparams[0].IP;
				param["EPC_PORT"]=dataparams[0].PORT;
			  	$.post("${ctx}/epc/configuration/createTrafficInfos.action", param, function(data){
					 doSearchUrl('apn_traffic_table_list', param,'${ctx}/epc/configuration/getTrafficInfos.action');
			  	});
			  	$("#selectedApn_auto_confirm").datagrid('loadData',{total:0,rows:[]});
			  	$("#apn_apn_name").val("");
		  	}
	  	}else{
	  		showPCRFAddFlag = true;
		  	$(hearderClass).children().removeClass("addToeGW");
		  	$("."+dclass).slideUp(500);
		  	$(hearderClass).removeAttr("onclick");
		  	$(hearderClass).attr("onclick","operEpcConfiguration('ON','"+dclass+"')");
		  	$(hearderClass+" .titleButtonText").text("<%=rb.getString("TianJia")%>");
			$(hearderClass+" .circleBg").addClass("add_circle");
			$(hearderClass+" .circleBg").removeClass("close_circle");
		  	$("."+dclass+" input[type='text']").val("");
		  	$("."+dclass+" .prompt").text("");
		  	//$(".epcConfigInfoItemDiv span.prompt").text("");
		  	
	  	}
  	
 	}

  
  	function modifyTrafficConfiguration(indx){
	  	//初始化数据 
	  
	  	$("#addeGWTraffic .titleButtonText").html("<%=rb.getString("GuanBi")%>");
		$("#addeGWTraffic .circleBg").removeClass("add_circle");
		$("#addeGWTraffic .circleBg").addClass("close_circle");
		//解除click事件
	  	$("#addeGWTraffic").removeAttr("onclick");
	  	//重新绑定
	  	$("#addeGWTraffic").attr("onclick","closeModifyTrafficSetting()");
	  	var row=$("#traffic_table").datagrid('getData').rows[indx];
	  	$(".modifyTrafficSetting #trafficCpeName").text(row.EPC_NAME+"["+row.EPC_SERVER_IP+"]");
	  	$("#m_PCC_NAME").val(row.PCC_NAME);
	  	$("#m_qci").val(row.QCI);
	  	$("#m_ARP_PL").val(row.ARP_PL);
	  	//$("#m_ARP_PCI").val(row.ARP_PCI);
	  	//$("#m_ARP_PVI").val(row.ARP_PVI);
	  	$("#m_ARP_PCI").combobox('setValues',row.ARP_PCI);
	  	$("#m_ARP_PVI").combobox('setValues',row.ARP_PVI);
	  	$("#m_PRECEDENCE").val(row.PRECEDENCE);
	  	$("#m_mbrUL").val(row.MBR_UL);
	  	$("#m_mbrDL").val(row.MBR_DL);
	  	$("#m_gbrUL").val(row.GBR_UL);
	  	$("#m_gbrDL").val(row.GBR_DL);
	  	setGBR_m(row.QCI);
	  	var dl = $("#selectedmRule_auto_confirm");
	  	$("#selectedmRule_auto_confirm").datagrid('loadData',{total:0,rows:[]});
	  	if(row.PF_LIST.length!=0){
	  		var arrpcc = row.PF_LIST.split(",");
	  		for(var i=0;i<arrpcc.length;i++){
	  			var row = {"PF_ID": arrpcc[i]};//, "APP_NAME": arrpcc[i]
	   			dl.datagrid("appendRow", row); 
	  		}
	  	}
	  	var epc_id = $("#PCRFStting .shuntChooseTit li span").attr("epc_id");
	    var epc_ip = $("#PCRFStting .shuntChooseTit li span").attr("title");
	    var epc_port = $("#PCRFStting .shuntChooseTit li span").attr("port");
	    var param = {};
	    param["EPC_ID"]=epc_id;
	    param["EPC_IP"]=epc_ip;
	    param["EPC_PORT"]=epc_port;
	    $.post("${ctx}/epc/configuration/createFilterInfos.action", param, function(data){
			 doSearchUrl('m_pcc_rule_table_list', param,'${ctx}/epc/configuration/getFilterInfos.action');
	  	});
	  	$(".modifyTrafficSetting").slideDown(500);
	  	
  	}
	
  	function modifyFilterConfiguration(indx){
  		//addeGWFilter
  		$("#addeGWFilter .titleButtonText").html("<%=rb.getString("GuanBi")%>");
		$("#addeGWFilter .circleBg").removeClass("add_circle");
		$("#addeGWFilter .circleBg").addClass("close_circle");
		//解除click事件
	  	$("#addeGWFilter").removeAttr("onclick");
	  	//重新绑定
	  	$("#addeGWFilter").attr("onclick","closeModifyFilterSetting()");
  		getProtocolCombobox();
	  	//初始化数据 
	  	var row=$("#filter_table").datagrid('getData').rows[indx];
	  	var pf_id=row.PF_ID;
	  	$(".modifyFilterSetting #filterCpeName").text(row.NAME+"["+row.EPC_SERVER_IP+"]");
	  	//$(".modifyFilterSetting #filterCpeName").append('<div class="titleIcon_close iconSize" style="position:absolute;right:25px;top:27px;" onclick="closeModifyFilterSetting()"></div>');
	  	$("#m_pfID").val(row.PF_ID);
	  	$("#m_filterAPP_NAME").val(row.APP_NAME);
	  	$("#m_protocol").combobox("setValue",row.PROTOCOL);
	  	$("#m_filterIpMask").val(row.IP_MASK);
	  	$("#m_filterPort").val(row.PORT);
	  	$(".modifyFilterSetting").slideDown(300);
  	}
  
	function modifyImsiTrafficConfiguration(indx){
		$("#addeGWIMSITRAFFIC .titleButtonText").html("<%=rb.getString("GuanBi")%>");
		$("#addeGWIMSITRAFFIC .circleBg").removeClass("add_circle");
		$("#addeGWIMSITRAFFIC .circleBg").addClass("close_circle");
		//解除click事件
	  	$("#addeGWIMSITRAFFIC").removeAttr("onclick");
	  	//重新绑定
	  	$("#addeGWIMSITRAFFIC").attr("onclick","closeModifyImsiTrafficSetting()");
		//初始化数据 
	  	var row=$("#epc_spr_imsi_traffic_table").datagrid('getData').rows[indx];
		$(".modifyImsiTrafficSetting #trafficCpeName").text(row.EPC_NAME+"["+row.EPC_SERVER_IP+"]");
	  	$("#m_IMSI").val(row.IMSI);
	  	$("#m_APN_NAME").val(row.APN_NAME);
	  	var dl = $("#selectedmimsi_auto_confirm");
	  	$("#selectedmimsi_auto_confirm").datagrid('loadData',{total:0,rows:[]});
	  	if(row.PCC_LIST.length!=0){
	  		var arrpcc = row.PCC_LIST.split(",");
	  		for(var i=0;i<arrpcc.length;i++){
	  			var row = {"PCC_NAME": arrpcc[i]};
	   			dl.datagrid("appendRow", row); 
	  		}
	  	}
	  	var epc_id = $("#PCRFStting .shuntChooseTit li span").attr("epc_id");
	    var epc_ip = $("#PCRFStting .shuntChooseTit li span").attr("title");
	    var epc_port = $("#PCRFStting .shuntChooseTit li span").attr("port");
	    var param = {};
	    param["EPC_ID"]=epc_id;
	    param["EPC_IP"]=epc_ip;
	    param["EPC_PORT"]=epc_port;
	    $.post("${ctx}/epc/configuration/createTrafficInfos.action", param, function(data){
			 doSearchUrl('m_imsi_traffic_table_list', param,'${ctx}/epc/configuration/getTrafficInfos.action');
	    });
	  	$(".modifyImsiTrafficSetting").slideDown(300);
	}  
	
	function modifyApnTrafficConfiguration(indx){
		$("#addeGWAPNTRAFFIC .titleButtonText").html("<%=rb.getString("GuanBi")%>");
		$("#addeGWAPNTRAFFIC .circleBg").removeClass("add_circle");
		$("#addeGWAPNTRAFFIC .circleBg").addClass("close_circle");
		//解除click事件
	  	$("#addeGWAPNTRAFFIC").removeAttr("onclick");
	  	//重新绑定
	  	$("#addeGWAPNTRAFFIC").attr("onclick","closeModifyApnTrafficSetting()");
		
		//初始化数据 
	  	var row=$("#epc_spr_apn_traffic_table").datagrid('getData').rows[indx];
		$(".modifyApnTrafficSetting #trafficCpeName").text(row.EPC_NAME+"["+row.EPC_SERVER_IP+"]");
	  	$("#mm_APN_NAME").val(row.APN_NAME);
	  	var dl = $("#selectedmApn_auto_confirm");
	  	$("#selectedmApn_auto_confirm").datagrid('loadData',{total:0,rows:[]});
	  	if(row.PCC_LIST.length!=0){
	  		var arrpcc = row.PCC_LIST.split(",");
	  		for(var i=0;i<arrpcc.length;i++){
	  			var row = {"PCC_NAME": arrpcc[i]};
	   			dl.datagrid("appendRow", row); 
	  		}
	  	}
	  	var epc_id = $("#PCRFStting .shuntChooseTit li span").attr("epc_id");
		var epc_ip = $("#PCRFStting .shuntChooseTit li span").attr("title");
		var epc_port = $("#PCRFStting .shuntChooseTit li span").attr("port");
		var param = {};
		param["EPC_ID"]=epc_id;
		param["EPC_IP"]=epc_ip;
		param["EPC_PORT"]=epc_port;
		$.post("${ctx}/epc/configuration/createTrafficInfos.action", param, function(data){
				 doSearchUrl('mapn_traffic_table_list', param,'${ctx}/epc/configuration/getTrafficInfos.action');
		});
	  	$(".modifyApnTrafficSetting").slideDown(300);
	}
  
  	function deleteFilterConfiguration(indx){
		//初始化数据 
		var row=$("#filter_table").datagrid('getData').rows[indx];
		$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueRenShanChu")%>", function(r) {
		  if (r) {			  
			  savingCover();
			  $.post("${ctx}/epc/configuration/delSprFilterByID.action", {EPC_ID:row.EPC_ID,PF_ID:row.PF_ID,PROTOCOL:row.PROTOCOL,IP:row.IP_MASK,PORT:row.PORT,ADDR:row.EPC_SERVER_IP,ADDR_PORT:row.EPC_PORT,APP_NAME:row.APP_NAME,ID:row.ID},function(data){
				  cancelSavingCover();
				  if (data["success"]) {
		        	  showMsg('success_msg',"<%=rb.getString("ShanChuChengGong")%>");
		        	  $("#filter_table").datagrid("reload");
		        	  $("#traffic_table").datagrid("reload");
		          } else {
		        	  showMsg('error_msg',data["message"]);
		          }
		      }, "json");
		  }
	    }).addClass("seriousConfirm");
  	}
  	
  	function deleteImsiTrafficSetting(indx){
  		//初始化数据 
		var row=$("#epc_spr_imsi_traffic_table").datagrid('getData').rows[indx];
		$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueRenShanChu")%>", function(r) {
		  if (r) {			  
			  savingCover();
			  $.post("${ctx}/epc/configuration/deleteGwImsiTraffic.action", {IMSI:row.IMSI,APN_NAME:row.APN_NAME,PCC_LIST:row.PCC_LIST,EPC_ID:row.EPC_ID,EPC_SERVER_IP:row.EPC_SERVER_IP,EPC_PORT:row.EPC_PORT},function(data){
				  cancelSavingCover();
				  if (data["success"]) {
		        	  showMsg('success_msg',"<%=rb.getString("ShanChuChengGong")%>");
		        	  $("#epc_spr_imsi_traffic_table").datagrid("reload");
		          } else {
		        	  showMsg('error_msg',data["message"]);
		          }
		      }, "json");
		  }
	    }).addClass("seriousConfirm");
  	}
  	
  	function deleteApnTrafficSetting(indx){
  		//初始化数据 
		var row=$("#epc_spr_apn_traffic_table").datagrid('getData').rows[indx];
		$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueRenShanChu")%>", function(r) {
		  if (r) {			  
			  savingCover();
			  $.post("${ctx}/epc/configuration/deleteGwApnTraffic.action", {APN_NAME:row.APN_NAME,PCC_LIST:row.PCC_LIST,EPC_ID:row.EPC_ID,EPC_SERVER_IP:row.EPC_SERVER_IP,EPC_PORT:row.EPC_PORT},function(data){
				  cancelSavingCover();
				  if (data["success"]) {
		        	  showMsg('success_msg',"<%=rb.getString("ShanChuChengGong")%>");
		        	  $("#epc_spr_apn_traffic_table").datagrid("reload");
		          } else {
		        	  showMsg('error_msg',data["message"]);
		          }
		      }, "json");
		  }
	    }).addClass("seriousConfirm");
  	}
  	
  	function deleteTrafficSetting(indx){
	  	var row=$("#traffic_table").datagrid('getData').rows[indx];
	  	var req={};
	  	req["PCC_NAME"]=row.PCC_NAME;
	  	req["QCI"]=row.QCI;
	  	req["ARP_PL"]=row.ARP_PL;
	  	req["ARP_PCI"]=row.ARP_PCI;
	  	req["ARP_PVI"]=row.ARP_PVI;
	  	req["MBR_UL"]=row.MBR_UL;
	  	req["MBR_DL"]=row.MBR_DL;
	  	req["GBR_UL"]=row.GBR_UL;
	  	req["GBR_DL"]=row.GBR_DL;
	  	req["PRECEDENCE"]=row.PRECEDENCE;
	  	req["ADDR"]=row.EPC_SERVER_IP;
	  	req["ADDR_PORT"]=row.EPC_PORT;
	  	req["EPC_ID"]=row.EPC_ID;
	  
	  	$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueRenShanChu")%>", function(r) {
		  if (r) {			  
			  savingCover();
			  $.post("${ctx}/epc/configuration/delSprTrafficByID.action",req,function(data){
				  cancelSavingCover();
		          if (data["success"]) {
		        	  showMsg('success_msg',"<%=rb.getString("ShanChuChengGong")%>");
		        	  $("#traffic_table").datagrid("reload");
		          } else {
		        	  showMsg('error_msg',data["message"]);
		          }
		      }, "json");
		  }
	    }).addClass("seriousConfirm");
  	}
  	
  	function closeModifyTrafficSetting(){
  		$("#addeGWTraffic .titleButtonText").html("<%=rb.getString("TianJia")%>");
		$("#addeGWTraffic .circleBg").addClass("add_circle");
		$("#addeGWTraffic .circleBg").removeClass("close_circle");
		//解除click事件
	  	$("#addeGWTraffic").removeAttr("onclick");
	  	//重新绑定
	  	$("#addeGWTraffic").attr("onclick","operEpcConfiguration('ON','addTrafficSetting')");
	  	var row=$("#traffic_table").datagrid('getSelected');
	  	$("#m_PCC_NAME").val(row.PCC_NAME);
	  	$("#m_qci").val(row.QCI);
	  	$("#m_ARP_PL").val(row.ARP_PL);
	  	//$("#m_ARP_PCI").val(row.ARP_PCI);
	  	//$("#m_ARP_PVI").val(row.ARP_PVI);
	  	$("#m_ARP_PCI").combobox('setValues',row.ARP_PCI);
	  	$("#m_ARP_PVI").combobox('setValues',row.ARP_PVI);
	  	$("#m_PRECEDENCE").val(row.PRECEDENCE);
	  	$("#m_mbrUL").val(row.MBR_UL);
	  	$("#m_mbrDL").val(row.MBR_DL);
	  	$("#m_gbrUL").val(row.GBR_UL);
	  	$("#m_gbrDL").val(row.GBR_DL);
	  	$(".modifyTrafficSetting .epcConfigTrafficDiv span[class='prompt']").text("");
	  	$(".modifyTrafficSetting").slideUp(300);
	  	setGBR_m(row.QCI);
  	}
  	
  	function closeModifyImsiTrafficSetting(){
  		$("#addeGWIMSITRAFFIC .titleButtonText").html("<%=rb.getString("TianJia")%>");
		$("#addeGWIMSITRAFFIC .circleBg").addClass("add_circle");
		$("#addeGWIMSITRAFFIC .circleBg").removeClass("close_circle");
		//解除click事件
	  	$("#addeGWIMSITRAFFIC").removeAttr("onclick");
	  	//重新绑定
	  	$("#addeGWIMSITRAFFIC").attr("onclick","operEpcConfiguration('ON','addImsiTrafficSetting')");
  		var row=$("#epc_spr_imsi_traffic_table").datagrid('getSelected');
  	  	$("#m_IMSI").val(row.IMSI);
  	  	$("#m_APN_NAME").val(row.APN_NAME);
  	  	$(".modifyImsiTrafficSetting .epcConfigTrafficDiv span[class='prompt']").text("");
  	  	$(".modifyImsiTrafficSetting").slideUp(300);
  	}
  	
  	function closeModifyApnTrafficSetting(){
  		$("#addeGWAPNTRAFFIC .titleButtonText").html("<%=rb.getString("TianJia")%>");
		$("#addeGWAPNTRAFFIC .circleBg").addClass("add_circle");
		$("#addeGWAPNTRAFFIC .circleBg").removeClass("close_circle");
		//解除click事件
	  	$("#addeGWAPNTRAFFIC").removeAttr("onclick");
	  	//重新绑定
	  	$("#addeGWAPNTRAFFIC").attr("onclick","operEpcConfiguration('ON','addApnTrafficSetting')");
  		var row=$("#epc_spr_apn_traffic_table").datagrid('getSelected');
  	  	$("#mm_APN_NAME").val(row.APN_NAME);
  	  	$(".modifyApnTrafficSetting .epcConfigTrafficDiv span[class='prompt']").text("");
  	  	$(".modifyApnTrafficSetting").slideUp(300);
  	}
  	
  	function closeModifyFilterSetting(){
  		$("#addeGWFilter .titleButtonText").html("<%=rb.getString("TianJia")%>");
		$("#addeGWFilter .circleBg").addClass("add_circle");
		$("#addeGWFilter .circleBg").removeClass("close_circle");
		//解除click事件
	  	$("#addeGWFilter").removeAttr("onclick");
	  	//重新绑定
	  	$("#addeGWFilter").attr("onclick","operEpcConfiguration('ON','addFilterSetting')");
		var row=$("#filter_table").datagrid('getSelected');
	  	$("#m_filterIpMask").val(row.IP_MASK);
	  	$("#m_filterAPP_NAME").val(row.APP_NAME);
	  	$("#m_filterPort").val(row.PORT);
	  	$(".modifyFilterSetting .epcConfigInfoItemDiv span[class='prompt']").text("");
	  	$(".modifyFilterSetting").slideUp(300);
  	}
  

  	function rotateR(){
	  	var rBottomNew = $(".addTrafficSetting").position().top;
	  	if(rBottomNew <-800){
		  	$(".addTrafficSetting").animate({top:"0",opacity:"1"},500);
	  	}else{
		  	$(".addTrafficSetting").animate({top:"-900px",opacity:"0"},500);
	  	}
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
  /* 	原选择EPc的方法  需删除 */
  	/* function chooseShunt(id,name,addr,port,ele){
		$("#sprPacketFilter").val("");
		$("#sprCommonTraffic").val("");
		$("#sprImsiTraffic").val("");
		$("#sprApnTraffic").val("");
		$(".shuntChooseItem").slideUp(250);
		$(".chooseArrow").attr("src","${ctx}/css/images/bi/eGwArrowDown.png");
		var pfs=$(ele).attr("pfs");
		var param={};
		param["EPC_ID"]=id;
		param["EPC_IP"]=addr;
		param["EPC_PORT"]=port;
		$("#PCRFStting .shuntChooseTit li span").text(name);
		$("#PCRFStting .shuntChooseTit li span").attr("title",addr);
		$("#PCRFStting .shuntChooseTit li span").attr("port",port);
		$("#PCRFStting .shuntChooseTit li span").attr("epc_id",id);
		var tab=$("#tabs_service").tabs('getSelected');
		var index=$("#tabs_service").tabs('getTabIndex',tab);
		if(index == 0 && pfs==undefined){
			$("#select_epc_pf_val").text(name);
			savingCover();
			$.post("${ctx}/epc/configuration/createFilterInfos.action", param, function(data){
				 cancelSavingCover();
				 doSearchUrl('filter_table', param,'${ctx}/epc/configuration/getFilterInfos.action');
			});
		}else if(index ==1 && pfs==undefined){
			$("#select_epc_rule_val").text(name);
			savingCover();
			$.post("${ctx}/epc/configuration/createTrafficInfos.action", param, function(data){
				 cancelSavingCover();
				 doSearchUrl('traffic_table', param,'${ctx}/epc/configuration/getTrafficInfos.action');
			});
		}else if(index ==2 && pfs==undefined){
			$("#select_epc_imsi_val").text(name);
			savingCover();
			 $.post("${ctx}/epc/configuration/createImsiTraffic.action", param, function(data){
				 cancelSavingCover();
		  		 doSearchUrl('epc_spr_imsi_traffic_table', param,'${ctx}/epc/configuration/getImsiTrafficInfosPage.action');
			 }, "json");
		}else if(index ==3 && pfs==undefined){
			$("#select_epc_apn_val").text(name);
			savingCover();
			 $.post("${ctx}/epc/configuration/createApnTraffic.action", param, function(data){
				 cancelSavingCover();
				 doSearchUrl('epc_spr_apn_traffic_table', param,'${ctx}/epc/configuration/getApnTrafficInfosPage.action');
			 });
		}
		load_pf=0;
		load_rule=0;
		load_imsi=0;
		load_apn=0;
	} */
  
  	function verifyFilter(CLASS,PF_ID){
	  	var num=0;
	  	$.each($("."+CLASS+":checked"),function(index,ele){
		  	num++;
	  	});
	  	if(num>10){
		  	showMsg('prompt_msg',"<%=rb.getString("ZuiDaXuanZheTen")%>");
		  	var obj = document.getElementById(PF_ID);
		  	obj.checked=false;
	  	}
  	}
  
  	//添加epc过滤配置 
  	function addFilterConfiguration(){
  		
  	  	var appname=$("#filterAPP_NAME").val();
  	  	if(!checkRangLength("filterAPP_NAME",appname,1,64)){
		  	return;
	  	}
  	  	if(!checkName("filterAPP_NAME",appname)){
			return;
		}
  	  	/*
  	  	var reg = /^([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9_.-][a-zA-Z0-9]+)$/;
  	  	if(!reg.test(appname)){
  			$("#filterAPP_NAMECheckSpan").text("<%=rb.getString("ZhiNengShuRuZiMuShuZiXiaHuaXianXiaoShuDian")%>");
  		  	return;
  	  	}
  	  	*/
	  	var pcl=$("#protocol").combobox("getValue");
	  	var ip=$("#filterIpMask").val();
	  	if(!checkIpMaskFormat("filterIpMask",ip,pcl)){
		  	return;
	  	}
	  	//针对非IP，协议的ip为0.0.0.0时为空处理
	  	if(pcl != "ip" && ip.length>0 && ip.split("/")[0] == "0.0.0.0"){
		  	ip="";
	  	}
	  	var port=$("#filterPort").val();
	  	if(!checkPortOrRange("filterPort",port)){
		  	return;
	  	}
	  	var selectValue = $("#PCCchoseEPC").combobox("getValues");
	  	if(selectValue[0] == ""){
  			$('.prompt_msg').html("<%=rb.getString("QingXuanZheEPCFuWu")%>");
	  		var width = $('.prompt_msg').width();
	  		$('.prompt_msg').css("left",'50%');
	  		var left = parseFloat($('.prompt_msg').css("left")) - width/2;
	  		$('.prompt_msg').css("left",left+'px');
	  		$('.prompt_msg').animate({top:'55px'},200,function(){
	  			setTimeout(function(){
	  				$('.prompt_msg').animate({top:'-40px'},function(){
	  					$("#PCCchoseEPC").combobox("showPanel");
	  				})
	  			},3000)
	  		})
  			return;
  		}
		var paramsArray = $("#PCCchoseEPC").combobox("getData");
		//选中的参数
		var dataparams = paramsArray.filter(function(item){
				return item.EPC_ID == selectValue;
		});
		var addr = dataparams[0].IP;
		var epc_port = dataparams[0].PORT;
		var epc_id = dataparams[0].EPC_ID;
	  	$("#protocolCheckSpan").text("");
	  	if(checkPCRFPFConfig(epc_id,pcl,ip,port,'')>0){
		  	$("#protocolCheckSpan").text("<%=rb.getString("PROTOCOLIP_MASKDuanKouYiCunZai")%>");
		  	return;
	 	}
	  	savingCover();
	  	$.post("${ctx}/epc/configuration/addEpcFilter.action", {PROTOCOL:pcl,IP:ip,PORT:port,ADDR:addr,ADDR_PORT:epc_port,EPC_ID:epc_id,APP_NAME:appname}, function(data){
		  	cancelSavingCover();
          	if (data["success"]) {
        	  	showMsg('success_msg',"<%=rb.getString("ChengGong")%>");
        	  	operEpcConfiguration("OFF","addFilterSetting");
        	  	$("#filter_table").datagrid("reload");
          	} else {
        	  	showMsg('error_msg',"<%=rb.getString("CaoZuoShiBai")%>");
        	  	$("#filterPortCheckSpan").text(data["message"]);
          	}
      	}, "json");
  	}
  
  	function checkPCRFPFConfig(epc_id,PROTOCOL,IP_MASK,PORT,PF_ID){
  		var flag=0;
    	var request={};
    	request["EPC_ID"]=epc_id;
    	request["PROTOCOL"]=PROTOCOL;
    	request["IP_MASK"]=IP_MASK;
    	request["PORT"]=PORT;
    	request["PF_ID"]=PF_ID;
    	
    	$.ajax({
			type: "post",
			url: "${ctx}/epc/configuration/checkPCRFPFConfig.action",
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
  
  	//添加epc traffic配置 
  	function addTrafficConfiguration(){
  		//var epc_id=$("#PCRFStting .shuntChooseTit li span").attr("epc_id");
  		var selectValue = $("#PCCchoseEPC").combobox("getValues");
  		if(selectValue[0] == ""){
  			$('.prompt_msg').html("<%=rb.getString("QingXuanZheEPCFuWu")%>");
	  		var width = $('.prompt_msg').width();
	  		$('.prompt_msg').css("left",'50%');
	  		var left = parseFloat($('.prompt_msg').css("left")) - width/2;
	  		$('.prompt_msg').css("left",left+'px');
	  		$('.prompt_msg').animate({top:'55px'},200,function(){
	  			setTimeout(function(){
	  				$('.prompt_msg').animate({top:'-40px'},function(){
	  					$("#PCCchoseEPC").combobox("showPanel");
	  				})
	  			},3000)
	  		})
  			return;
  		}
		var paramsArray = $("#PCCchoseEPC").combobox("getData");
		//选中的参数
		var dataparams = paramsArray.filter(function(item){
				return item.EPC_ID == selectValue;
		});
		var addr = dataparams[0].IP;
		var epc_port = dataparams[0].PORT;
		var epc_id = dataparams[0].EPC_ID;
	  	var pccname=$("#Traffic_PCC_NAME").val();
	  	if(!checkRangLength("Traffic_PCC_NAME",pccname,1,40)){
	  		$("#Traffic_PCC_NAME").select().focus();
		  	return;
	  	}
	  	if(!checkName("Traffic_PCC_NAME",pccname)){
			return;
		}
	  	/*
	  	var reg = /^([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9_.-][a-zA-Z0-9]+)$/
  	  	if(!reg.test(pccname)){
  			$("#Traffic_PCC_NAMECheckSpan").text("<%=rb.getString("ZhiNengShuRuZiMuShuZiXiaHuaXianXiaoShuDian")%>");
  			$("#Traffic_PCC_NAME").select().focus();
  			return;
  	  	}
	  	*/
  	  	if(checkPccName(epc_id,pccname)>0){
  			$("#Traffic_PCC_NAMECheckSpan").text("<%=rb.getString("PCCMingChengYiCunZai")%>");
  			$("#Traffic_PCC_NAME").select().focus();
  			return;
  	  	}
	  	var qci=$("#qci").val();
	  	if(!checkIntRange("qci",qci,1,9)){
	  		$("#qci").select().focus();
		  	return;
	  	}
	  	var ARPPL=$("#ARP_PL").val();
	  	if(!checkIntRange("ARP_PL",ARPPL,1,15)){
		  	return;
	  	}
	  	//var ARPPCI=$("#ARP_PCI").val();
	  	//var ARPPVI=$("#ARP_PVI").val();
	  	var ARPPCI=$("#ARP_PCI").combobox('getValues')+"";
	  	var ARPPVI=$("#ARP_PVI").combobox('getValues')+"";
	  	var PRECEDENCE = $("#PRECEDENCE").val();
	  	if(!checkIntRange("PRECEDENCE",PRECEDENCE,0,255)){
		  	return;
	  	}
	  	var mbrUL=$("#mbrUL").val();
	  	if(!checkIntRange("mbrUL",mbrUL,1000,104857600)){
		  	return;
	  	}
	  	var mbrDL=$("#mbrDL").val();
	  	if(!checkIntRange("mbrDL",mbrDL,1000,104857600)){
		  	return;
	  	}
	  	var gbrUL=$("#gbrUL").val();
	  	if(qci>=1 && qci<=4 && !checkIntRange("gbrUL",gbrUL,1000,104857600)){
		  	return;
	  	}

	  	var gbrDL=$("#gbrDL").val();
	  	if(qci>=1 && qci<=4 && !checkIntRange("gbrDL",gbrDL,1000,104857600)){
		  	return;
	  	}

	  	//增加判断 GBR_UL<=MBR_UL,GBR_DL<=MBR_DL zss
	  	if(parseInt(gbrUL)>parseInt(mbrUL)){
		   	$("#gbrULCheckSpan").text("<%=rb.getString("QingShuRuXiaoYuHuoDengYuMBR_ULZhi")%>");
			return;
	   	}else{
		   	$("#gbrULCheckSpan").text("");
	   	}
	  	if(parseInt(gbrDL)>parseInt(mbrDL)){
		   	$("#gbrDLCheckSpan").text("<%=rb.getString("QingShuRuXiaoYuHuoDengYuMBR_DLZhi")%>");
			return;
	   	}else{
		   	$("#gbrDLCheckSpan").text("");
	  	}
	  	//var addr=$("#PCRFStting .shuntChooseTit li span").attr("title");
	  	//var epc_port=$("#PCRFStting .shuntChooseTit li span").attr("port");
	  <%-- 	if(addr==""){
		  	$.messager.alert(TiShi, "<%=rb.getString("QingXuanZheEPCFuWu")%>");
		  	//$("#PCRFStting .shuntChooseTit li").first().click();
		  	return;
	  	} --%>

	  	var dl = $("#selectedRule_auto_confirm");
		var rows = dl.datagrid("getRows");
		var selCellLength = rows.length;
		var needDelCellArr = new Array(); //需要删除的小站编码数组
		if (selCellLength > 0) {
			for (var i = 0; i < selCellLength; i++) {
				if (rows[i]["PF_ID"]) {
					needDelCellArr.push(rows[i]["PF_ID"]);
	    		}
			}
		}
		
	  	var req={};
	  	req["PCC_NAME"]=pccname;
	  	req["QCI"]=qci;
	  	req["ARP_PL"]=ARPPL;
	  	req["ARP_PCI"]=ARPPCI;
	  	req["ARP_PVI"]=ARPPVI;
	  	req["MBR_UL"]=mbrUL;
	  	req["MBR_DL"]=mbrDL;
	  	req["GBR_UL"]=gbrUL;
	  	req["GBR_DL"]=gbrDL;
	  	req["PRECEDENCE"]=PRECEDENCE;
	  	req["EPC_ID"]=epc_id;
	  	req["ADDR"]=addr;
	  	req["ADDR_PORT"]=epc_port;
	  
	  	req["FILTERS"]=needDelCellArr.join(",");
	  	
	  	savingCover();
	  	$.post("${ctx}/epc/configuration/addEpcTraffic.action", req, function(data){
		  	cancelSavingCover();
          	if (data["success"]) {
        	  	showMsg('success_msg',"<%=rb.getString("ChengGong")%>");
        	  	operEpcConfiguration("OFF","addTrafficSetting");
        	  	$("#traffic_table").datagrid("reload");
          	} else {
        	  	showMsg('error_msg',"<%=rb.getString("CaoZuoShiBai")%>");
        	  	$("#gbrDLCheckSpan").text(data["message"]);
          	}
      	}, "json");
  	}
  	
  	function checkPccName(epc_id,pccname){
  		var flag=0;
    	var request={};
    	request["EPC_ID"]=epc_id;
    	request["PCC_NAME"]=pccname;
    	$.ajax({
			type: "post",
			url: "${ctx}/epc/configuration/checkPccName.action",
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
  
  	//修改epc过滤配置
  	function updateFilterConfiguration(){
	  	var row=$("#filter_table").datagrid('getSelected');
	  	var pcl=$("#m_protocol").combobox("getValue");
	  	var ip=$("#m_filterIpMask").val();
	  	if(!checkIpMaskFormat("m_filterIpMask",ip,pcl)){
		  	return;
	  	}
	  	//针对非IP，协议的ip为0.0.0.0时为空处理
	  	if(pcl != "ip" && ip.length>0 && ip.split("/")[0] == "0.0.0.0"){
		  	ip="";
	  	}
	  	var appname=$("#m_filterAPP_NAME").val();
	  	if(!checkRangLength("m_filterAPP_NAME",appname,1,64)){
		  	return;
	  	}
	  	if(!checkName("m_filterAPP_NAME",appname)){
			return;
		}
	  	/*
	  	var reg = /^[a-zA-Z][a-zA-Z0-9_]+$/;
  	  	if(!reg.test(appname)){
  			$("#m_filterAPP_NAMECheckSpan").text("it format error.");
  		  	return;
  	  	}
  	  	*/
	  	var port=$("#m_filterPort").val();
	  	if(!checkPortOrRange("m_filterPort",port)){
		  	return;
	  	}
	  	$("#m_protocolCheckSpan").text("");
	  	if(checkPCRFPFConfig(row.EPC_ID,pcl,ip,port,row.PF_ID)>0){
		  	$("#m_protocolCheckSpan").text("<%=rb.getString("PROTOCOLIP_MASKDuanKouYiCunZai")%>");
		  	return;
	 	}
	  	if(row.PROTOCOL==$("#m_protocol").combobox("getValue") && row.IP_MASK==ip && row.PORT==port && appname==row.APP_NAME){//没做任何修改，关掉 
		  	$(".modifyFilterSetting").animate({right:'-1800px'},500);
	      	return;
	  	}
	  
	  	savingCover();
	  	$.post("${ctx}/epc/configuration/updateEpcFilter.action", {EPC_ID:row.EPC_ID,PF_ID:row.PF_ID,PROTOCOL:$("#m_protocol").combobox("getValue"),IP:ip,PORT:port,ADDR:row.EPC_SERVER_IP,ADDR_PORT:row.EPC_PORT,ID:row.ID,APP_NAME:appname}, function(data){
		  	cancelSavingCover();
          	if (data["success"]) {
        	  	showMsg('success_msg',"<%=rb.getString("ChengGong")%>");
        	  	$("#filter_table").datagrid("reload");
        	  	$(".modifyFilterSetting").slideUp(300);
        	  	$("#addeGWFilter .titleButtonText").html("<%=rb.getString("TianJia")%>");
        		$("#addeGWFilter .circleBg").addClass("add_circle");
        		$("#addeGWFilter .circleBg").removeClass("close_circle");
        		//解除click事件
        	  	$("#addeGWFilter").removeAttr("onclick");
        	  	//重新绑定
        	  	$("#addeGWFilter").attr("onclick","operEpcConfiguration('ON','addFilterSetting')");
          	} else {
        	  	showMsg('error_msg',"<%=rb.getString("CaoZuoShiBai")%>");
        	  	$("#m_filterPortCheckSpan").text(data["message"]);
          	}
      	}, "json");
  	}
  	//修改traffic配置 zss
  	function updateTrafficConfiguration(){
	  	var row=$("#traffic_table").datagrid('getSelected');
	  
	  	var m_PCC_NAME=$("#m_PCC_NAME").val();
	  	if(!checkRangLength("m_PCC_NAME",m_PCC_NAME,1,40)){
		  	return;
	  	}
	  	if(!checkName("m_PCC_NAME",m_PCC_NAME)){
			return;
		}
	  	/*
	  	var reg = /^[a-zA-Z][a-zA-Z0-9_]+$/;
  	  	if(!reg.test(m_PCC_NAME)){
  			$("#m_PCC_NAMECheckSpan").text("it format error.");
  		  	return;
  	  	}*/
	  	var qci=$("#m_qci").val();
	  	if(!checkIntRange("m_qci",qci,1,9)){
		  	return;
	  	}
	  	var m_ARP_PL=$("#m_ARP_PL").val();
	  	if(!checkIntRange("m_ARP_PL",m_ARP_PL,1,15)){
		  	return;
	  	}
	  	//var m_ARP_PCI=$("#m_ARP_PCI").val();
	  	//var m_ARP_PVI=$("#m_ARP_PVI").val();
	  	var m_ARP_PCI=$("#m_ARP_PCI").combobox('getValues')+"";
	  	var m_ARP_PVI=$("#m_ARP_PVI").combobox('getValues')+"";

	  	var mbrUL=$("#m_mbrUL").val();
	  	if(!checkIntRange("m_mbrUL",mbrUL,1000,104857600)){
		  	return;
	  	}
	  	var mbrDL=$("#m_mbrDL").val();
	  	if(!checkIntRange("m_mbrDL",mbrDL,1000,104857600)){
		  	return;
	  	}
	  	var gbrUL=$("#m_gbrUL").val();
	  	if(qci>=1 && qci<=4 && !checkIntRange("m_gbrUL",gbrUL,1000,104857600)){
		  	return;
	  	}

	  	var gbrDL=$("#m_gbrDL").val();
	  	if(qci>=1 && qci<=4 && !checkIntRange("m_gbrDL",gbrDL,1000,104857600)){
		  	return;
	  	}

	  	if(qci>=5 && qci<=9){
		  	gbrUL="";
		  	gbrDL="";
	  	}
	  	var m_PRECEDENCE=$("#m_PRECEDENCE").val();
	  	if(!checkIntRange("m_PRECEDENCE",m_PRECEDENCE,0,255)){
		  	return;
	  	}
	  	//增加判断 GBR_UL<=MBR_UL,GBR_DL<=MBR_DL zss
	  	if(parseInt(gbrUL)>parseInt(mbrUL)){
		   	$("#m_gbrULCheckSpan").text("<%=rb.getString("QingShuRuXiaoYuHuoDengYuMBR_ULZhi")%>");
			return;
	   	}else{
		   	$("#m_gbrULCheckSpan").text("");
	   	}
	  
	  	if(parseInt(gbrDL)>parseInt(mbrDL)){
		   	$("#m_gbrDLCheckSpan").text("<%=rb.getString("QingShuRuXiaoYuHuoDengYuMBR_DLZhi")%>");
			return;
	   	}else{
		   	$("#m_gbrDLCheckSpan").text("");
	  	}
	  	var dl = $("#selectedmRule_auto_confirm");
		var rows = dl.datagrid("getRows");
		var selCellLength = rows.length;
		var needDelCellArr = new Array(); //需要删除的小站编码数组
		if (selCellLength > 0) {
			for (var i = 0; i < selCellLength; i++) {
				if (rows[i]["PF_ID"]) {
					needDelCellArr.push(rows[i]["PF_ID"]);
	    		}
			}
		}
		var pfListArr = new Array();
		if(row.PF_LIST.length>0){
			pfListArr = row.PF_LIST.split(",");
		}
	  	//如果没做任何变化，关掉
	  	if(m_PCC_NAME==row.PCC_NAME && qci==row.QCI && m_PRECEDENCE==row.PRECEDENCE 
	  			&& m_ARP_PL==row.ARP_PL && m_ARP_PCI==row.ARP_PCI && m_ARP_PVI==row.ARP_PVI 
	  			&& mbrUL==row.MBR_UL && mbrDL==row.MBR_DL && gbrUL==row.GBR_UL && gbrDL==row.GBR_DL
	  			&& needDelCellArr.sort().toString()==pfListArr.sort().toString()){
		  	$(".modifyTrafficSetting").slideUp(300);
    	  	$("#addeGWTraffic .titleButtonText").html("<%=rb.getString("TianJia")%>");
    		$("#addeGWTraffic .circleBg").addClass("add_circle");
    		$("#addeGWTraffic .circleBg").removeClass("close_circle");
    		//解除click事件
    	  	$("#addeGWTraffic").removeAttr("onclick");
    	  	//重新绑定
    	  	$("#addeGWTraffic").attr("onclick","operEpcConfiguration('ON','addTrafficSetting')");
		  	return;
	  	}
	  	var request={};
	  	request["PCC_NAME"]=$("#m_PCC_NAME").val();
	  	request["QCI"]=$("#m_qci").val();
	  	request["ARP_PL"]=m_ARP_PL;
	  	request["ARP_PCI"]=m_ARP_PCI;
	  	request["ARP_PVI"]=m_ARP_PVI;
	  	request["MBR_UL"]=$("#m_mbrUL").val();
	  	request["MBR_DL"]=$("#m_mbrDL").val();
	  	request["GBR_UL"]=gbrUL;
	  	request["GBR_DL"]=gbrDL;
	  	request["PRECEDENCE"]=m_PRECEDENCE;
	  	request["EPC_ID"]=row.EPC_ID;
	  	request["ADDR"]=row.EPC_SERVER_IP;
	  	request["OLDPCC_NAME"]=row.PCC_NAME;
	  	request["ADDR_PORT"]=row.EPC_PORT;
	  	request["FILTER_ID"]=row.PF_LIST;
	  	request["FILTERS"]=needDelCellArr.join(",");
	  	savingCover();
	  	$.post("${ctx}/epc/configuration/updateEpcTraffic.action", request, function(data){
		  	cancelSavingCover();
          	if (data["success"]) {
        	  	showMsg('success_msg',"<%=rb.getString("ChengGong")%>");
        	  	$(".modifyTrafficSetting").slideUp(300);
        	  	$("#addeGWTraffic .titleButtonText").html("<%=rb.getString("TianJia")%>");
        		$("#addeGWTraffic .circleBg").addClass("add_circle");
        		$("#addeGWTraffic .circleBg").removeClass("close_circle");
        		//解除click事件
        	  	$("#addeGWTraffic").removeAttr("onclick");
        	  	//重新绑定
        	  	$("#addeGWTraffic").attr("onclick","operEpcConfiguration('ON','addTrafficSetting')");
        	  	$("#traffic_table").datagrid("reload");
          	} else {
        	  	showMsg('error_msg',"<%=rb.getString("CaoZuoShiBai")%>");
          	}
      	}, "json");
	  
  	}
  	function queryPacketFilter(){
  		var selectValue = $("#PCCchoseEPC").combobox("getValues");
  		if(selectValue[0] == ""){
  			$('.prompt_msg').html("<%=rb.getString("QingXuanZheEPCFuWu")%>");
	  		var width = $('.prompt_msg').width();
	  		$('.prompt_msg').css("left",'50%');
	  		var left = parseFloat($('.prompt_msg').css("left")) - width/2;
	  		$('.prompt_msg').css("left",left+'px');
	  		$('.prompt_msg').animate({top:'55px'},200,function(){
	  			setTimeout(function(){
	  				$('.prompt_msg').animate({top:'-40px'},function(){
	  					$("#PCCchoseEPC").combobox("showPanel");
	  				})
	  			},3000)
	  		})
  			return;
  		}else{
  			var paramsArray = $("#PCCchoseEPC").combobox("getData");
  			//选中的参数
  			var dataparams = paramsArray.filter(function(item){
  					return item.EPC_ID == selectValue;
  			});
  			var addr = dataparams[0].IP;
  			var epc_port = dataparams[0].PORT;
  			var epc_id = dataparams[0].EPC_ID;
  	  		var sprPacketFilter = $("#sprPacketFilter").val();
  		  	var param={};
  		  	param["EPC_ID"]=epc_id;
  			param["ADDR"]=addr;
  			param["ADDR_PORT"]=epc_port;
  			param["APP_NAME"]=sprPacketFilter;
  		  	doSearchUrl('filter_table', param,'${ctx}/epc/configuration/getFilterInfos.action');
  		}
		
  	}
  	
  	function queryPCCRULEFilter(){
  	  var selectValue = $("#PCCchoseEPC").combobox("getValues");
		if(selectValue[0] == ""){
			$('.prompt_msg').html("<%=rb.getString("QingXuanZheEPCFuWu")%>");
	  		var width = $('.prompt_msg').width();
	  		$('.prompt_msg').css("left",'50%');
	  		var left = parseFloat($('.prompt_msg').css("left")) - width/2;
	  		$('.prompt_msg').css("left",left+'px');
	  		$('.prompt_msg').animate({top:'55px'},200,function(){
	  			setTimeout(function(){
	  				$('.prompt_msg').animate({top:'-40px'},function(){
	  					$("#PCCchoseEPC").combobox("showPanel");
	  				})
	  			},3000)
	  		})
			return;
		}else{
			var paramsArray = $("#PCCchoseEPC").combobox("getData");
			//选中的参数
			var dataparams = paramsArray.filter(function(item){
					return item.EPC_ID == selectValue;
			});
			var addr = dataparams[0].IP;
			var epc_port = dataparams[0].PORT;
			var epc_id = dataparams[0].EPC_ID;
	  		var sprPacketFilter = $("#sprPacketFilter").val();
	  		var param={};
		  	param["EPC_ID"]=epc_id;
			param["ADDR"]=addr;
			param["ADDR_PORT"]=epc_port;
			param["PCC_NAME"]=$("#sprCommonTraffic").val();
		  	doSearchUrl('traffic_table', param,'${ctx}/epc/configuration/getTrafficInfos.action');
		}
	  	
  	}
  	
  	function queryImsiTraffic(){
  		var selectValue = $("#PCCchoseEPC").combobox("getValues");
		if(selectValue[0] == ""){
			$('.prompt_msg').html("<%=rb.getString("QingXuanZheEPCFuWu")%>");
	  		var width = $('.prompt_msg').width();
	  		$('.prompt_msg').css("left",'50%');
	  		var left = parseFloat($('.prompt_msg').css("left")) - width/2;
	  		$('.prompt_msg').css("left",left+'px');
	  		$('.prompt_msg').animate({top:'55px'},200,function(){
	  			setTimeout(function(){
	  				$('.prompt_msg').animate({top:'-40px'},function(){
	  					$("#PCCchoseEPC").combobox("showPanel");
	  				})
	  			},3000)
	  		})
			return;
		}else{
			var paramsArray = $("#PCCchoseEPC").combobox("getData");
			//选中的参数
			var dataparams = paramsArray.filter(function(item){
					return item.EPC_ID == selectValue;
			});
			var addr = dataparams[0].IP;
			var epc_port = dataparams[0].PORT;
			var epc_id = dataparams[0].EPC_ID;
	  		var sprPacketFilter = $("#sprPacketFilter").val();
	  		var param={};
	  		var IMSI = $("#sprImsiTraffic").val();
	  		param["EPC_ID"]=epc_id;
			param["IMSI"]=IMSI;
	  		doSearchUrl('epc_spr_imsi_traffic_table', param,'${ctx}/epc/configuration/getImsiTrafficInfosPage.action');
		}		
  	}
  	
  	function queryApnTraffic(){
  		var selectValue = $("#PCCchoseEPC").combobox("getValues");
		if(selectValue[0] == ""){
			$('.prompt_msg').html("<%=rb.getString("QingXuanZheEPCFuWu")%>");
	  		var width = $('.prompt_msg').width();
	  		$('.prompt_msg').css("left",'50%');
	  		var left = parseFloat($('.prompt_msg').css("left")) - width/2;
	  		$('.prompt_msg').css("left",left+'px');
	  		$('.prompt_msg').animate({top:'55px'},200,function(){
	  			setTimeout(function(){
	  				$('.prompt_msg').animate({top:'-40px'},function(){
	  					$("#PCCchoseEPC").combobox("showPanel");
	  				})
	  			},3000)
	  		})
			return;
		}else{
			var paramsArray = $("#PCCchoseEPC").combobox("getData");
			//选中的参数
			var dataparams = paramsArray.filter(function(item){
					return item.EPC_ID == selectValue;
			});
			var addr = dataparams[0].IP;
			var epc_port = dataparams[0].PORT;
			var epc_id = dataparams[0].EPC_ID;
	  		var sprPacketFilter = $("#sprPacketFilter").val();
	  		var param={};
	  		var APN_NAME = $("#sprApnTraffic").val();
	  		param["EPC_ID"]=epc_id;
			param["APN_NAME"]=APN_NAME;
	   	 	doSearchUrl('epc_spr_apn_traffic_table', param,'${ctx}/epc/configuration/getApnTrafficInfosPage.action');
		}		
   	}
  	
  	//检查长度
  	function checkRangLength(id,value,minLength,maxLength){
      if(value=="" || value.length<minLength || value.length>maxLength){
		$("#"+id).focus().select();
		$("#"+id+"CheckSpan").text("<%=rb.getString("QingShuRuDeMingChenZai")%> {"+minLength+"} <%=rb.getString("AND")%> {"+maxLength+"} <%=rb.getString("ChangDu")%>");
		return false;
	  }else{
		$("#"+id+"CheckSpan").text("");
		return true;
	  }
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
  	
  	//校验IP
  	function isValidIP(ip){
	  var reg =  /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/     
	  return reg.test(ip);     
  	}
  	function checkIpAddressFormat(id,value){
	  if(!isValidIP(value)){
	     $("#"+id).focus().select();
	     $("#"+id+"CheckSpan").text("<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>");
	     return false;
	  }else{
	     $("#"+id+"CheckSpan").text("");
	     return true;
	  }
  	}
  	//校验端口号 范围 0-65535
  	function checkPort(id,value){
		if(isNumeric(value)&& parseInt(value)>=0 && parseInt(value)<=65535){
			$("#"+id+"CheckSpan").text("");
			return true;
		}else{
			//$("#"+id).focus().select();
	   		$("#"+id+"CheckSpan").text("<%=rb.getString("QingShuRuYouXiaoDuanKou")%>");
	    	return false;
		}
  	}
  
  	//校验端口号 范围 1-65535 同时支持输入端口范围,用'-'分开 
  	function checkPortOrRange(id,value){
	
		if(value==""){
			$("#"+id+"CheckSpan").text("");
			return true;
		}  
		var pos=value.split("-");
	
		for(var i=0;i<pos.length;i++){
		
			if(!isNumeric(pos[i]) || parseInt(pos[i])<1 || parseInt(pos[i])>65535){
				//$("#"+id).focus().select();
		    	$("#"+id+"CheckSpan").text("<%=rb.getString("QingShuRuYouXiaoDuanKou")%>");
		    	return false;
			}else{
				$("#"+id+"CheckSpan").text("");
			}
		}
		return true;
  	}
  
	//检查IP/MASK格式 
	function checkIpMaskFormat(id,value,pval){
		var pos=value.split("/");
		if(pos.length==0 || pos.length>2 || (pval == "ip" && value.length==0)){//空或者格式不正确
			//$("#"+id).focus().select();
			$("#"+id+"CheckSpan").text("<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>/MASK(example:192.168.9.20/24 or 192.168.9.20)");
			return false;
		}else{
			$("#"+id+"CheckSpan").text("");
		}
		for(var i=0;(i<pos.length)&&(value.length>0) ;i++){
			if(i==0){//校验是否是正确的IP 
				if(!isValidIP(pos[i])){
			    //$("#"+id).focus().select();
			    $("#"+id+"CheckSpan").text("<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>/MASK(example:192.168.9.20/24 or 192.168.9.20)");
			    return false;
			 }else if(pos[i]=="0.0.0.0" && pval=="ip"){
				  $("#"+id+"CheckSpan").text("<%=rb.getString("IPBuNengWei")%>");
				  return false;
			 }else{
				  $("#"+id+"CheckSpan").text("");
			 }
		 }
		 if(i==1){//mask[1-32]
			 if(isNumeric(pos[i])&& parseInt(pos[i])>=1 && parseInt(pos[i])<=32){
				  $("#"+id+"CheckSpan").text("");
			 }else{
				  //$("#"+id).focus().select();
				  $("#"+id+"CheckSpan").text("<%=rb.getString("QingShuRuZhengQueDeYanMaFanWei")%>");
				  return false;
			 }
		 }
		}
		return true;
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
	
	//检查整数范围
	function checkIntRange(id,value,minLength,maxLength){
		if(isNumeric(value)&& parseInt(value)>=minLength && parseInt(value)<=maxLength){
			$("#"+id+"CheckSpan").text("");
			return true;
		}else{
			$("#"+id).focus().select();
			$("#"+id+"CheckSpan").text("<%=rb.getString("QingShuRuYouXiaoDeZhi")%>("+minLength+"~"+maxLength+").");
			return false;
		}
	}
	
	function checkLongRange(id,value,Length){
		if(isNumeric(value) && value.length==Length){
			$("#"+id+"CheckSpan").text("");
			return true;
		}else{
			$("#"+id).focus().select();
			$("#"+id+"CheckSpan").text("<%=rb.getString("QingShuRuYouXiaoDeZhi")%> "+Length+" <%=rb.getString("ShuZi")%>.");
		}
		return false;
	}
  
  	var delCellCodeObjArr = new Array();//存储基站列表中需要删除的基站信息
	//选择小站-向右键头的点击事件
	function addSelectedCell_auto_confirm(table,list) {
		var selCell = $("#"+table).datagrid("getSelections");
		var selCellLength = selCell.length;
		if (selCellLength == 0) {
			return;
		}
		var newSelectdArr = new Array();
		var dl = $("#"+list);
		var rows = dl.datagrid("getRows");
		if(rows.length>=10||(selCellLength+rows.length)>10){
			showMsg('prompt_msg',"<%=rb.getString("ZuiDaXuanZheTen")%>");
			return;
		}
		for (var selCellCount = 0; selCellCount < selCellLength; selCellCount++) {
			var i = 0;
			var cellName = "";
			if("pcc_rule_table_list"==table||"m_pcc_rule_table_list"==table){
				cellName = selCell[selCellCount]["PF_ID"];
			}else{
				cellName = selCell[selCellCount]["PCC_NAME"];
			}
			
			for (; i < rows.length; i++) {
				//如果当前已经选择了该小站，跳出循环
	    		if (rows[i]["PF_ID"] == cellName || rows[i]["PCC_NAME"] == cellName) {
	    			break;
	    		}
	    	}
			//i == rows.length表示当前没有选择该小站，需要加到右侧列表中
			if (i == rows.length) {
				var row = "";
				if("pcc_rule_table_list"==table||"m_pcc_rule_table_list"==table){
					row = {"PF_ID": selCell[selCellCount]["PF_ID"]};//, "APP_NAME": selCell[selCellCount]["APP_NAME"]
				}else{
					row = {"PCC_NAME": selCell[selCellCount]["PCC_NAME"]};
				}
	   			dl.datagrid("appendRow", row); 
	   			//newSelectdArr.push(selectedCellObj);
			}
		}
		/*
		if (rows.length >= 2 && (rows[0]["PF_ID"].length == 0 || rows[0]["APP_NAME"] == "<%=rb.getString("QingXuanZe")%>")) {
			dl.datagrid("deleteRow", 0);
		}*/
		$("#"+table).datagrid("uncheckAll");
	}
  	
	//删除已选中的小站-向左键头的点击事件
	function delSelectedCell_auto_confirm(tablelist) {
		var dl = $("#"+tablelist);
		var selCell = dl.datalist("getSelections");
		var selCellLength = selCell.length;
		var needDelCellArr = new Array(); //需要删除的小站编码数组
		if (selCellLength > 0) {
			for (var selCellCount = 0; selCellCount < selCell.length; selCellCount++) {
				if(tablelist=="selectedRule_auto_confirm"||tablelist=="selectedmRule_auto_confirm"){
					if (selCell[selCellCount]["PF_ID"]) {
						needDelCellArr.push(selCell[selCellCount]);
		    		}
				}else{
					if (selCell[selCellCount]["PCC_NAME"]) {
						needDelCellArr.push(selCell[selCellCount]);
		    		}
				}
			}
	    	if (needDelCellArr.length > 0) {
	    		//从已选基站列表中删除基站
	    		for(var cellCount = 0; cellCount < needDelCellArr.length; cellCount++) {
	    			var rowIndex = $("#"+tablelist).datalist("getRowIndex", needDelCellArr[cellCount]);
	    			$("#"+tablelist).datalist("deleteRow",rowIndex);
	    		}
	    	}
			var rows = dl.datalist("getRows");
			/*
			if (rows.length == 0) {
				var row = {index: 0, row: {value: "", text: "<%=rb.getString("QingXuanZe")%>"}};
				dl.datalist("insertRow", row);
			}*/
			//将复选框取消选中
			$("#"+tablelist).datalist("uncheckAll");
		}
	}
	
	function addImsiTrafficConfiguration(){
		var IMSI=$("#IMSI").val();
		if(!checkLongRange("IMSI",IMSI,15)){
			return;
		}
		var APN_NAME=$("#APN_NAME").val();
		if(!checkRangLength("APN_NAME",APN_NAME,1,100)){
			return;
		}
		if(!checkName("APN_NAME",APN_NAME)){
			return;
		}
		/*
		var reg = /^([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9_.-][a-zA-Z0-9]+)$/
		if(!reg.test(APN_NAME)){
			$("#APN_NAMECheckSpan").text("<%=rb.getString("ZhiNengShuRuZiMuShuZiXiaHuaXianXiaoShuDian")%>");
			return;
		}
		*/
		var dl = $("#selectedImsi_auto_confirm");
		var rows = dl.datagrid("getRows");
		var selCellLength = rows.length;
		var needDelCellArr = new Array(); //需要删除的小站编码数组
		if (selCellLength > 0) {
			for (var i = 0; i < selCellLength; i++) {
				if (rows[i]["PCC_NAME"]) {
					needDelCellArr.push(rows[i]["PCC_NAME"]);
	    		}
			}
		}
		/*
		if(needDelCellArr.length==0){
			$.messager.alert(TiShi, "SELECT PCC_LIST");
			return;
		}
		*/
		/* var addr=$("#PCRFStting .shuntChooseTit li span").attr("title");
		var epc_port=$("#PCRFStting .shuntChooseTit li span").attr("port");
		var epc_id=$("#PCRFStting .shuntChooseTit li span").attr("epc_id"); */
		var selectValue = $("#PCCchoseEPC").combobox("getValues");
		if(selectValue[0] == ""){
  			$('.prompt_msg').html("<%=rb.getString("QingXuanZheEPCFuWu")%>");
	  		var width = $('.prompt_msg').width();
	  		$('.prompt_msg').css("left",'50%');
	  		var left = parseFloat($('.prompt_msg').css("left")) - width/2;
	  		$('.prompt_msg').css("left",left+'px');
	  		$('.prompt_msg').animate({top:'55px'},200,function(){
	  			setTimeout(function(){
	  				$('.prompt_msg').animate({top:'-40px'},function(){
	  					$("#PCCchoseEPC").combobox("showPanel");
	  				})
	  			},3000)
	  		})
  			return;
  		}
		var paramsArray = $("#PCCchoseEPC").combobox("getData");
		//选中的参数
		var dataparams = paramsArray.filter(function(item){
				return item.EPC_ID == selectValue;
		});
		var addr = dataparams[0].IP;
		var epc_port = dataparams[0].PORT;
		var epc_id = dataparams[0].EPC_ID;
		<%-- if(addr==""){
			$.messager.alert(TiShi, "<%=rb.getString("QingXuanZheEPCFuWu")%>");
			//$("#PCRFStting .shuntChooseTit li").first().click();
			return;
		} --%>
		$("#APN_NAMECheckSpan").text("");
		if(checkImsiApnName(epc_id,IMSI,APN_NAME)>0){
			$("#APN_NAMECheckSpan").text("<%=rb.getString("IMSIAPN_NAMEYiCunZai")%>");
			return;
		}
		var request={};
		request["IMSI"]=IMSI;
		request["APN_NAME"]=APN_NAME;
		request["EPC_ID"]=epc_id;
		request["ADDR"]=addr;
		request["ADDR_PORT"]=epc_port;
		request["PCC_LIST"]=needDelCellArr.join(",");
		savingCover();
		$.post("${ctx}/epc/configuration/addGwImsiTraffic.action", request, function(data){
			cancelSavingCover();
			if (data["success"]) {
				showMsg('success_msg',"<%=rb.getString("ChengGong")%>");
				$("#addeGWIMSITRAFFIC .titleButtonText").html("<%=rb.getString("TianJia")%>");
				$("#addeGWIMSITRAFFIC .circleBg").addClass("add_circle");
				$("#addeGWIMSITRAFFIC .circleBg").removeClass("close_circle");
				//解除click事件
			  	$("#addeGWIMSITRAFFIC").removeAttr("onclick");
			  	//重新绑定
			  	$("#addeGWIMSITRAFFIC").attr("onclick","operEpcConfiguration('ON','addImsiTrafficSetting')");
				$(".addImsiTrafficSetting").slideUp(500);
				$("#epc_spr_imsi_traffic_table").datagrid("reload");
			}else{
				showMsg('error_msg',"<%=rb.getString("CaoZuoShiBai")%>");
			}
		}, "json");
	}
	
	function checkImsiApnName(epc_id,imsi,apn_name){
  		var flag=0;
    	var request={};
    	request["EPC_ID"]=epc_id;
    	request["IMSI"]=imsi;
    	request["APN_NAME"]=apn_name;
    	$.ajax({
			type: "post",
			url: "${ctx}/epc/configuration/checkImsiApnName.action",
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
	
	function updateImsiTrafficConfiguration(){
		var row=$("#epc_spr_imsi_traffic_table").datagrid('getSelected');
		  
		var IMSI=$("#m_IMSI").val();
		/*
		if(!checkRangLength("m_IMSI",IMSI,15,15)){
			return;
		}*/
		var APN_NAME=$("#m_APN_NAME").val();
		if(!checkRangLength("m_APN_NAME",APN_NAME,1,100)){
			return;
		}
		/*
		var reg = /^[a-zA-Z][a-zA-Z0-9_]+$/;
		if(!reg.test(APN_NAME)){
			$("#m_APN_NAMECheckSpan").text("it format error.");
			return;
		}*/
		var dl = $("#selectedmimsi_auto_confirm");
		var rows = dl.datagrid("getRows");
		var selCellLength = rows.length;
		var needDelCellArr = new Array(); 
		if (selCellLength > 0) {
			for (var i = 0; i < selCellLength; i++) {
				if (rows[i]["PCC_NAME"]) {
					needDelCellArr.push(rows[i]["PCC_NAME"]);
	    		}
			}
		}
		/*
		if(needDelCellArr.length==0){
			$.messager.alert(TiShi, "select PCC_LIST");
			return;
		}*/
		var request={};
		request["IMSI"]=IMSI;
		request["APN_NAME"]=APN_NAME;
		request["PCC_LIST"]=needDelCellArr.join(",");
		request["EPC_ID"]=row.EPC_ID;
		request["EPC_SERVER_IP"]=row.EPC_SERVER_IP;
		request["EPC_PORT"]=row.EPC_PORT;
		request["OLDIMSI"]=row.IMSI;
		request["OLDAPN_NAME"]=row.APN_NAME;
		savingCover();
		$.post("${ctx}/epc/configuration/updateGwImsiTraffic.action", request, function(data){
			cancelSavingCover();
			if (data["success"]) {
				showMsg('success_msg',"<%=rb.getString("ChengGong")%>");
				$(".modifyImsiTrafficSetting").slideUp(300);
				$("#addeGWIMSITRAFFIC .titleButtonText").html("<%=rb.getString("TianJia")%>");
				$("#addeGWIMSITRAFFIC .circleBg").addClass("add_circle");
				$("#addeGWIMSITRAFFIC .circleBg").removeClass("close_circle");
				//解除click事件
			  	$("#addeGWIMSITRAFFIC").removeAttr("onclick");
			  	//重新绑定
			  	$("#addeGWIMSITRAFFIC").attr("onclick","operEpcConfiguration('ON','addImsiTrafficSetting')");
				$("#epc_spr_imsi_traffic_table").datagrid("reload");
			}else{
				showMsg('error_msg',"<%=rb.getString("CaoZuoShiBai")%>");
			}
		}, "json");
	}
	
	//
	function addApnTrafficConfiguration(){
		var APN_NAME=$("#apn_apn_name").val();
		if(!checkRangLength("apn_apn_name",APN_NAME,1,100)){
			return;
		}
		if(!checkName("apn_apn_name",APN_NAME)){
			return;
		}
		/*
		$("#apn_apn_nameCheckSpan").text("");
		var reg = /^([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9_.-][a-zA-Z0-9]+)$/
		if(!reg.test(APN_NAME)){
			$("#apn_apn_nameCheckSpan").text("<%=rb.getString("ZhiNengShuRuZiMuShuZiXiaHuaXianXiaoShuDian")%>");
			return;
		}
		*/
		var dl = $("#selectedApn_auto_confirm");
		var rows = dl.datagrid("getRows");
		var selCellLength = rows.length;
		var needDelCellArr = new Array(); 
		if (selCellLength > 0) {
			for (var i = 0; i < selCellLength; i++) {
				if (rows[i]["PCC_NAME"]) {
					needDelCellArr.push(rows[i]["PCC_NAME"]);
	    		}
			}
		}
		/*
		if(needDelCellArr.length==0){
			$.messager.alert(TiShi, "SELECT PCC_LIST");
			return;
		}
		*/
		/* var addr=$("#PCRFStting .shuntChooseTit li span").attr("title");
		var epc_port=$("#PCRFStting .shuntChooseTit li span").attr("port");
		var epc_id=$("#PCRFStting .shuntChooseTit li span").attr("epc_id"); */
		var selectValue = $("#PCCchoseEPC").combobox("getValues");
		if(selectValue[0] == ""){
  			$('.prompt_msg').html("<%=rb.getString("QingXuanZheEPCFuWu")%>");
	  		var width = $('.prompt_msg').width();
	  		$('.prompt_msg').css("left",'50%');
	  		var left = parseFloat($('.prompt_msg').css("left")) - width/2;
	  		$('.prompt_msg').css("left",left+'px');
	  		$('.prompt_msg').animate({top:'55px'},200,function(){
	  			setTimeout(function(){
	  				$('.prompt_msg').animate({top:'-40px'},function(){
	  					$("#PCCchoseEPC").combobox("showPanel");
	  				})
	  			},3000)
	  		})
  			return;
  		}
		var paramsArray = $("#PCCchoseEPC").combobox("getData");
		//选中的参数
		var dataparams = paramsArray.filter(function(item){
				return item.EPC_ID == selectValue;
		});
		var addr = dataparams[0].IP;
		var epc_port = dataparams[0].PORT;
		var epc_id = dataparams[0].EPC_ID;
	  	var pccname=$("#Traffic_PCC_NAME").val();
		<%-- if(addr==""){
			$.messager.alert(TiShi, "<%=rb.getString("QingXuanZheEPCFuWu")%>");
			$("#PCRFStting .shuntChooseTit li").first().click();
			return;
		} --%>
		if(checkApnNameTraffic(epc_id,APN_NAME)>0){
			$("#apn_apn_nameCheckSpan").text("<%=rb.getString("APNMingChengYiCunZai")%>");
			return;
		}
		var request={};
		request["APN_NAME"]=APN_NAME;
		request["EPC_ID"]=epc_id;
		request["ADDR"]=addr;
		request["ADDR_PORT"]=epc_port;
		request["PCC_LIST"]=needDelCellArr.join(",");
		savingCover();
		$.post("${ctx}/epc/configuration/addGwApnTraffic.action", request, function(data){
			cancelSavingCover();
			if (data["success"]) {
				showMsg('success_msg',"<%=rb.getString("ChengGong")%>");
				$("#addeGWAPNTRAFFIC .titleButtonText").html("<%=rb.getString("TianJia")%>");
				$("#addeGWAPNTRAFFIC .circleBg").addClass("add_circle");
				$("#addeGWAPNTRAFFIC .circleBg").removeClass("close_circle");
				//解除click事件
			  	$("#addeGWAPNTRAFFIC").removeAttr("onclick");
			  	//重新绑定
			  	$("#addeGWAPNTRAFFIC").attr("onclick","operEpcConfiguration('ON','addApnTrafficSetting')");
			  	$(".addApnTrafficSetting").slideUp(500);
			  	$("#epc_spr_apn_traffic_table").datagrid("reload");
			}else{
				showMsg('error_msg',"<%=rb.getString("CaoZuoShiBai")%>");
			}
		}, "json");
	}
	
	function checkApnNameTraffic(epc_id,apn_name){
  		var flag=0;
    	var request={};
    	request["EPC_ID"]=epc_id;
    	request["APN_NAME"]=apn_name;
    	$.ajax({
			type: "post",
			url: "${ctx}/epc/configuration/checkApnNameTraffic.action",
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
	
	function updateApnTrafficConfiguration(){
		var row=$("#epc_spr_apn_traffic_table").datagrid('getSelected');
		  
		var APN_NAME=$("#mm_APN_NAME").val();
		if(!checkRangLength("mm_APN_NAME",APN_NAME,1,100)){
			return;
		}
		/*
		var reg = /^[a-zA-Z][a-zA-Z0-9_]+$/;
		if(!reg.test(APN_NAME)){
			$("#mm_APN_NAMECheckSpan").text("it format error.");
			return;
		}*/
		var dl = $("#selectedmApn_auto_confirm");
		var rows = dl.datagrid("getRows");
		var selCellLength = rows.length;
		var needDelCellArr = new Array(); 
		if (selCellLength > 0) {
			for (var i = 0; i < selCellLength; i++) {
				if (rows[i]["PCC_NAME"]) {
					needDelCellArr.push(rows[i]["PCC_NAME"]);
	    		}
			}
		}
		/*
		if(needDelCellArr.length==0){
			$.messager.alert(TiShi, "select PCC_LIST");
			return;
		}
		*/
		var request={};
		request["APN_NAME"]=APN_NAME;
		request["PCC_LIST"]=needDelCellArr.join(",");
		request["EPC_ID"]=row.EPC_ID;
		request["EPC_SERVER_IP"]=row.EPC_SERVER_IP;
		request["EPC_PORT"]=row.EPC_PORT;
		request["OLDAPN_NAME"]=row.APN_NAME;
		
		savingCover();
		$.post("${ctx}/epc/configuration/updateGwApnTraffic.action", request, function(data){
			cancelSavingCover();
			if (data["success"]) {
				showMsg('success_msg',"<%=rb.getString("ChengGong")%>");
				$(".modifyApnTrafficSetting").slideUp(300);
				$("#addeGWAPNTRAFFIC .titleButtonText").html("<%=rb.getString("TianJia")%>");
				$("#addeGWAPNTRAFFIC .circleBg").addClass("add_circle");
				$("#addeGWAPNTRAFFIC .circleBg").removeClass("close_circle");
				//解除click事件
			  	$("#addeGWAPNTRAFFIC").removeAttr("onclick");
			  	//重新绑定
			  	$("#addeGWAPNTRAFFIC").attr("onclick","operEpcConfiguration('ON','addApnTrafficSetting')");
				$("#epc_spr_apn_traffic_table").datagrid("reload");
			}else{
				showMsg('error_msg',"<%=rb.getString("CaoZuoShiBai")%>");
			}
		}, "json");
	}
	
	function serialPccName(){
		var param={};
		//var epc_id=$("#PCRFStting .shuntChooseTit li span").attr("epc_id");
		var epcId =  $("#PCCchoseEPC").combobox('getValue');
	  	if((epcId==undefined||epcId=="") && tag=="ON"){
	  		$('.prompt_msg').html("<%=rb.getString("QingXuanZheEPCFuWu")%>");
	  		var width = $('.prompt_msg').width();
	  		$('.prompt_msg').css("left",'50%');
	  		var left = parseFloat($('.prompt_msg').css("left")) - width/2;
	  		$('.prompt_msg').css("left",left+'px');
	  		$('.prompt_msg').animate({top:'55px'},200,function(){
	  			setTimeout(function(){
	  				$('.prompt_msg').animate({top:'-40px'},function(){
	  					$("#PCCchoseEPC").combobox("showPanel");
	  				})
	  			},3000)
	  		})
		 	return;
	  	}
		var query_imsi_pcc_name = $("#query_imsi_pcc_name").val();
		param["PCC_NAME"]=query_imsi_pcc_name;
		param["EPC_ID"]=epcId;
		doSearchUrl('imsi_traffic_table_list', param,'${ctx}/epc/configuration/getTrafficInfos.action');
	}
	
	function serialPccName_m(){
		var param={};
		var epcId =  $("#PCCchoseEPC").combobox('getValue');
	  	if((epcId==undefined||epcId=="") && tag=="ON"){
	  		$('.prompt_msg').html("<%=rb.getString("QingXuanZheEPCFuWu")%>");
	  		var width = $('.prompt_msg').width();
	  		$('.prompt_msg').css("left",'50%');
	  		var left = parseFloat($('.prompt_msg').css("left")) - width/2;
	  		$('.prompt_msg').css("left",left+'px');
	  		$('.prompt_msg').animate({top:'55px'},200,function(){
	  			setTimeout(function(){
	  				$('.prompt_msg').animate({top:'-40px'},function(){
	  					$("#PCCchoseEPC").combobox("showPanel");
	  				})
	  			},3000)
	  		})
		 	return;
	  	}
		//var epc_id=$("#PCRFStting .shuntChooseTit li span").attr("epc_id");
		var query_imsi_pcc_name = $("#query_imsi_pcc_name_m").val();
		param["PCC_NAME"]=query_imsi_pcc_name;
		param["EPC_ID"]=epcId;
		doSearchUrl('m_imsi_traffic_table_list', param,'${ctx}/epc/configuration/getTrafficInfos.action');
	}
	
	function serialApnPccName(){
		var param={};
		//var epc_id=$("#PCRFStting .shuntChooseTit li span").attr("epc_id");
		var epcId =  $("#PCCchoseEPC").combobox('getValue');
	  	if((epcId==undefined||epcId=="") && tag=="ON"){
	  		$('.prompt_msg').html("<%=rb.getString("QingXuanZheEPCFuWu")%>");
	  		var width = $('.prompt_msg').width();
	  		$('.prompt_msg').css("left",'50%');
	  		var left = parseFloat($('.prompt_msg').css("left")) - width/2;
	  		$('.prompt_msg').css("left",left+'px');
	  		$('.prompt_msg').animate({top:'55px'},200,function(){
	  			setTimeout(function(){
	  				$('.prompt_msg').animate({top:'-40px'},function(){
	  					$("#PCCchoseEPC").combobox("showPanel");
	  				})
	  			},3000)
	  		})
		 	return;
	  	}
		var query_apn_pcc_name = $("#query_apn_pcc_name").val();
		param["PCC_NAME"]=query_apn_pcc_name;
		param["EPC_ID"]=epcId;
		doSearchUrl('apn_traffic_table_list', param,'${ctx}/epc/configuration/getTrafficInfos.action');
	}
	
	function serialApnPccName_m(){
		var param={};
		var epcId =  $("#PCCchoseEPC").combobox('getValue');
	  	if((epcId==undefined||epcId=="") && tag=="ON"){
	  		$('.prompt_msg').html("<%=rb.getString("QingXuanZheEPCFuWu")%>");
	  		var width = $('.prompt_msg').width();
	  		$('.prompt_msg').css("left",'50%');
	  		var left = parseFloat($('.prompt_msg').css("left")) - width/2;
	  		$('.prompt_msg').css("left",left+'px');
	  		$('.prompt_msg').animate({top:'55px'},200,function(){
	  			setTimeout(function(){
	  				$('.prompt_msg').animate({top:'-40px'},function(){
	  					$("#PCCchoseEPC").combobox("showPanel");
	  				})
	  			},3000)
	  		})
		 	return;
	  	}		var query_apn_pcc_name = $("#query_apn_pcc_name_m").val();
		param["PCC_NAME"]=query_apn_pcc_name;
		param["EPC_ID"]=epcId;
		doSearchUrl('mapn_traffic_table_list', param,'${ctx}/epc/configuration/getTrafficInfos.action');
	}
	
	function serialPCRFPF_ID(){
		var param={};
		var epcId =  $("#PCCchoseEPC").combobox('getValue');
	  	if((epcId==undefined||epcId=="") && tag=="ON"){
	  		$('.prompt_msg').html("<%=rb.getString("QingXuanZheEPCFuWu")%>");
	  		var width = $('.prompt_msg').width();
	  		$('.prompt_msg').css("left",'50%');
	  		var left = parseFloat($('.prompt_msg').css("left")) - width/2;
	  		$('.prompt_msg').css("left",left+'px');
	  		$('.prompt_msg').animate({top:'55px'},200,function(){
	  			setTimeout(function(){
	  				$('.prompt_msg').animate({top:'-40px'},function(){
	  					$("#PCCchoseEPC").combobox("showPanel");
	  				})
	  			},3000)
	  		})
		 	return;
	  	}
		var query_pcrf_pf_id = $("#query_pcrf_pf_id").val();
		param["PF_ID"]=query_pcrf_pf_id;
		param["EPC_ID"]=epcId;
		//param["EPC_ID"]=epc_id;
		doSearchUrl('pcc_rule_table_list', param,'${ctx}/epc/configuration/getFilterInfos.action');
	}
	
	function serialPCRFPF_ID_m(){
		var param={};
		//var epc_id=$("#PCRFStting .shuntChooseTit li span").attr("epc_id");
		var epcId =  $("#PCCchoseEPC").combobox('getValue');
	  	if((epcId==undefined||epcId=="") && tag=="ON"){
	  		$('.prompt_msg').html("<%=rb.getString("QingXuanZheEPCFuWu")%>");
	  		var width = $('.prompt_msg').width();
	  		$('.prompt_msg').css("left",'50%');
	  		var left = parseFloat($('.prompt_msg').css("left")) - width/2;
	  		$('.prompt_msg').css("left",left+'px');
	  		$('.prompt_msg').animate({top:'55px'},200,function(){
	  			setTimeout(function(){
	  				$('.prompt_msg').animate({top:'-40px'},function(){
	  					$("#PCCchoseEPC").combobox("showPanel");
	  				})
	  			},3000)
	  		})
		 	return;
	  	}
		var query_pcrf_pf_id = $("#query_pcrf_pf_id_m").val();
		param["PF_ID"]=query_pcrf_pf_id;
		param["EPC_ID"]=epcId;
		doSearchUrl('m_pcc_rule_table_list', param,'${ctx}/epc/configuration/getFilterInfos.action');
	}
	
	function getPF_ListFormatter(value, rowData, rowIndex){
		var res="";
		/*
		if(value!=""){
			var request={};
			request["EPC_ID"]=rowData.EPC_ID;
			request["PF_LIST"]=rowData.PF_LIST;
			$.ajax({
				type: "post",
				url: "${ctx}/epc/configuration/getPF_LISTToPccName.action",
				data: request,
				async: false,
				dataType:"json",
				success: function(data) {
					res = data["message"];
				}
			});
		}else{
			res=value;
		}*/
		return value;
	}
	
	function getDisabledFormatter(value, rowData, rowIndex){
		if(value=="0"){
			return "disabled";
		}else if(value=="1"){
			return "enable";
		}else{
			return value;
		}
	}
	
	function getProtocolCombobox(){
		$("#protocol,#m_protocol").empty();
		var PROTOCOL_INIT=[];
		$.ajax({
			type: "post",
			url: "${ctx}/epc/configuration/getEpcSprPccPfProtocolList.action",
			async: false,
			dataType:"json",
			success: function(data) {
				$.each(data,function(idx,obj){
					 var PROTOCOL_LIST ={text:obj.PROTOCOL_TYPE,value:obj.PROTOCOL_ID};
					 PROTOCOL_INIT.push(PROTOCOL_LIST);
				});
			}
		});
		
		$("#protocol,#m_protocol").combobox({
			valueField:'value',
			textField:'text',
			data:PROTOCOL_INIT,
			onSelect:function(obj){}
		});
	}
	
	
	
	//QCI大于等于5时 GBR不可输入
	function setGBR(value){
		if(value>=5){
			$("#gbrUL,#gbrDL").val("").attr("disabled","disabled");
		}else{
			$("#gbrUL,#gbrDL").removeAttr("disabled");
		}
	}
	
	function setGBR_m(value){
		if(value>=5){
			$("#m_gbrUL,#m_gbrDL").val("").attr("disabled","disabled");
		}else{
			$("#m_gbrUL,#m_gbrDL").removeAttr("disabled");
		}
	}
	
	function loadSuccess(){
		var tb = $(this);
		setTimeout(function(){tb.datagrid("resize")},300)
		$(this).datagrid("fixRownumber");
    	$(this).datagrid("enableContextmenuAutoSize");
		$(this).datagrid("getPager").pagination({
			showRefresh:false,
			showPageList:false,
			layout:['list']
		})
	}
	//选择EPC
	function chooseShunt(){
		$("#sprPacketFilter").val("");
		$("#sprCommonTraffic").val("");
		$("#sprImsiTraffic").val("");
		$("#sprApnTraffic").val("");
		var selectValue = $("#PCCchoseEPC").combobox("getValues");
		var paramsArray = $("#PCCchoseEPC").combobox("getData");
		//选中的参数
		var dataparams = paramsArray.filter(function(item){
				return item.EPC_ID == selectValue;
		});
		var param = {};
		param["EPC_ID"]=dataparams[0].EPC_ID;
		param["EPC_IP"]=dataparams[0].IP;
		param["EPC_PORT"]=dataparams[0].PORT;
		//selectedEPCItem 判断选择的是哪个epc的项目
		switch (selectedEPCItem){
			case 0:
				savingCover();
				$.post("${ctx}/epc/configuration/createFilterInfos.action", param, function(data){
					 cancelSavingCover();
					 doSearchUrl('filter_table', param,'${ctx}/epc/configuration/getFilterInfos.action');
				});
			break;
			
			case 1:
				savingCover();
				$.post("${ctx}/epc/configuration/createTrafficInfos.action", param, function(data){
					 cancelSavingCover();
					 doSearchUrl('traffic_table', param,'${ctx}/epc/configuration/getTrafficInfos.action');
				});
			break;
			case 2:
				savingCover();
				 $.post("${ctx}/epc/configuration/createImsiTraffic.action", param, function(data){
					 cancelSavingCover();
			  		 doSearchUrl('epc_spr_imsi_traffic_table', param,'${ctx}/epc/configuration/getImsiTrafficInfosPage.action');
				 }, "json");
			break;
			case 3:
				savingCover();
				 $.post("${ctx}/epc/configuration/createApnTraffic.action", param, function(data){
					 cancelSavingCover();
					 doSearchUrl('epc_spr_apn_traffic_table', param,'${ctx}/epc/configuration/getApnTrafficInfosPage.action');
				 });
			break;
		}
		load_pf=0;
		load_rule=0;
		load_imsi=0;
		load_apn=0;
	}
	
	//添加任务 与修改任务的加载前函数 将ecp_id 传过去
	function onBeforeladtaskTable(param){
		var epcId =  $("#PCCchoseEPC").combobox('getValue');
		param['EPC_ID'] = epcId;
	}
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
</script>
