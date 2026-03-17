<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<!-- 告警过滤新建模板开发页面…… -->
<style>
#filterInfoPage{
	display:flex;
	padding:10px 0px 0px 30px;
	height:100%;
	box-sizing:border-box;
	overflow:auto;
	position: relative;
}
#filterInfoPage .el-tabs__content {
	border: 1px solid #E9E9E9;
}
.headContent{
	font-size:14px;
	color:#0F344D;
	font-weight:700;
	margin-top: 20px;
	margin-bottom:10px;
}
.headContent img{
	vertical-align:top;
	margin-right:14px;
}
#filterInfoPage .el-input__inner, #filterInfoPage .el-textarea__inner{
	-webkit-appearance: none;
	-moz-appearance:textfield;
	box-sizing: border-box;
	outline: none;
	transition: border-color .2s cubic-bezier(.645,.045,.355,1);
	width: 100%;
}
.boxBorderCon{
	width:100%;
	height:27px;
	padding-top:10px;
}
.bottomline{
	width:80%;
	height:2px;
	background:#DCECF7;
	margin-top:35px;
}

#filterInfoPage .advanceQuery{
	padding-left:0px !important;
}
#filterInfoPage .el-select .el-input .el-select__caret{
	line-height:28px;
}
.boxContainerAlarmType .el-checkbox{
	width:180px;
}
.demonstration{
	display:block;
	margin-bottom:10px;
}
.alarmSpan{
	display:inline-block;
	height:21px;
	line-height:24px;
	margin-left:8px;
}
#filterInfoPage .el-icon-search{
	margin-left:0px !important;
} 
.boxBorderCon .el-icon-circle-close{
	line-height:20px;
}
#filterInfoPage .el-form-item{
	margin-left:34px;
}
#filterInfoPage .concentBoxENB ,#filterInfoPage .concentBoxUPS ,#filterInfoPage .concentBoxCPE {
	position: relative;
}
#filterInfoPage .concentBoxENB .el-form-item,#filterInfoPage .concentBoxUPS .el-form-item,#filterInfoPage .concentBoxCPE .el-form-item,#filterInfoPage .concentBoxGSM .el-form-item{
	margin-left:20px;
	margin-bottom: 10px;
}
#filterInfoPage .el-form{
	width:100%;
}
#filterInfoPage .el-form-pagrid-container .el-form{
	width: 425px;
}
#filterInfoPage .el-form-item__content{
	width:50%;
	min-width:500px;
	white-space: nowrap;
}
#filterInfoPage .el-form-table-container .el-form-item__content{
	width:100%;
}
#filterInfoPage .el-form-pagrid-container .el-form-item__content{
	width:98%;
}
#filterInfoPage .el-ctable-toolbar .el-input__inner{
	border:none; 
}
.queryInfo{
	display:inline-block;
	margin-right:35px;
}
.queryInfo label{
	display:block;
	line-height:24px;
}
#filterInfoPage .queryInfo .el-input__inner{
	border:1px solid #DEDFE6;
}
.concentBoxENB .el-pairgrid-title,.concentBoxUPS .el-pairgrid-title,.concentBoxCPE .el-pairgrid-title,.concentBoxGSM .el-pairgrid-title{
	top: -25px;
}
#filterInfoPage .templateNameStys div:first-child{
	overflow: hidden;
}
#filterInfoPage .templateNameStys .el-form-item__error{
	font-size: 12px;
    line-height: 1;
    padding-top: 0px;
    position: absolute;
    top: 34px;
    left: 0;
}
#filterInfoPage .el-form-item{
	margin-left:34px;
	margin-bottom: 10px;
}
#filterInfoPage .pairgrid-left  div:first-child .el-ctable{
	border-right: 1px solid #E9E9E9;
}
.emailPrompt{
	font-size: 12px;
	color: #BBBBBB;
	position: relative;
	display: inline-block;
	top:-8px;
}
.emailPrompt .el-icon:before{
	color: #BBBBBB;
	font-size: 12px;
}
#filterInfoPage .el-date-editor .el-range__close-icon{
	line-height: 26px;
}
#filterInfoPage .ContentClass{
	padding-bottom: 20px;
}
.alarmBottomLine{
	background-color:#E9E9E9;
	width: calc(100% + 30px);
	height: 1px;
	margin-left: -30px;
}
.alarmMinor .el-icon:before{
	color: #FFDA41;
	font-size: 16px;
}
.alarmMajor .el-icon:before{
	color: #FF973E;
	font-size: 16px;
}
.alarmCritical .el-icon:before{
	color: #FC5959;
	font-size: 16px;
}
.alarmWarning .el-icon:before{
	color: #60BEFC;
	font-size: 16px;
}
.ContentClass .el-tabs--card >.el-tabs__header{
	border-bottom:none;
}
.ContentClass .el-tabs__item{
	font-size: 14px;
	font-weight: normal;
}
#filterInfoPage .el-ctable .el-query .advanceQuery .el-input__inner{
	padding-left: 10px;
}

#filterInfoPage .allAlarmClass{
	position: absolute;
	left: 80px;
	top:2px;
}
#filterInfoPage .transition-box{
	z-index: 102!important;
}
</style> 
<div id="filterInfoPage">
	<el-form :model="ruleForm" :rules="rules" ref = "ruleForm" label-position="left" label-width="164" :hide-required-asterisk='true'>
		<div class="ContentClass">
			<!-- 基本信息 -->
			<div class="headContent">
				<img src="${ctx}/css/images/global/settingBetter.png"><%=rb.getString("JiBenXinXi") %>
			</div>
			<!--模板名称-->
			<el-form-item label="<%=rb.getString("GuoLvMoBanMingCheng")%>" prop="ruleName" class="templateNameStys" style="margin-bottom:0px">
				<el-input ref="" :disabled="viewFlag || defaultVal == '1'" v-model="ruleForm.ruleName" style="width:400px;padding-top:5px;"></el-input>
				<div class="emailPrompt"><span class="el-icon-circle-info el-icon" style="margin-right:10px;" ></span><span><%=rb.getString("MingChengChangDu")%>1~50</span></div>
			</el-form-item>
			<!--状态-->
			<el-form-item label="<%=rb.getString("ZhuangTai")%>" prop="status" style="margin-bottom:0px">
				<el-radio-group v-model='ruleForm.status' :disabled='viewFlag' style="margin-top:12px">
						<el-radio label="1" ><%=rb.getString("QiYong") %></el-radio>
						<el-radio label="0" ><%=rb.getString("JinYong") %></el-radio>
				</el-radio-group>
			</el-form-item>
			<!--执行动作-->
			<el-form-item label="<%=rb.getString("ZhiXingDongZuo")%>" prop="ruleType" style="margin-bottom:0px">
				<el-radio-group v-model='ruleForm.ruleType' :disabled='viewFlag' style="margin-top:12px">
						<el-radio label="1" ><%=rb.getString("BuRuKuBuXianShi")%></el-radio>
						<el-radio label="3" ><%=rb.getString("XianShiZiDongQueRen") %></el-radio>
				</el-radio-group>
			</el-form-item>
		</div>
		<div class="alarmBottomLine"></div>
		<div class="ContentClass">
			<!-- 设备选择 -->
			<div class="headContent">
				<img src="${ctx}/css/images/global/settingBetter.png"><%=rb.getString("TiaoJianSheZhi") %>
			</div>
			<!-- 设备类型 -->
			<el-form-item prop='deviceTypeList' label='<%=rb.getString("XinGaoJingYuan")%>' v-if="defaultVal !== '1'">
				<div class="boxBorderCon">
					<el-checkbox-group v-model="ruleForm.deviceTypeList" :disabled='viewFlag'  @change="deviceTypeListChange">
						<el-checkbox v-for=" items in initAlarmTypeArr" :key="items.value" :label="items.value">{{items.label}}</el-checkbox>
					</el-checkbox-group>
				</div>
			</el-form-item>
			<!-- 告警源选项卡  -->
			<el-tabs v-model="alarmTabsValue" type="card" :closable="!viewFlag" @tab-remove="removeTab" style="margin:0px 34px 10px 34px;"  v-if="alarmTabsShow == true && defaultVal !== '1'">
				<el-tab-pane label="ENB" name="ENB"  v-if="deviceTypeShowENB == true">
					<div class="concentBoxENB">
						<!-- ENB设备选择 --><!-- <%=rb.getString("SheBeiXuanZe")%> -->
						<el-form-item  prop='enbDeviceByDeviceGroup' label='' label-width="0px" :class="ruleForm.enbDeviceByDeviceGroup == 1 ? 'radioButtonSelect' : ''"> 
								<el-radio-group v-model='ruleForm.enbAllAndCustomize' :disabled='viewFlag' style="padding-top:17px;" @change="enbAllAndCustomizeChange">
									<el-radio :label="0" ><%=rb.getString("QuanXuan")%></el-radio>
									<el-radio :label="1" style="margin-left:50px"><%=rb.getString("DingZhi")%></el-radio>
								</el-radio-group>
								<el-radio-group v-model="ruleForm.enbDeviceByDeviceGroup" @change="enbDeviceByDeviceGroupChange"  :disabled='ruleForm.enbAllAndCustomize == 0 || viewFlag' size="mini"  style="padding-top:15px;padding-left:5px;">
										<el-radio-button :label="0"><%=rb.getString("KPISheBei")%></el-radio-button>
										<el-radio-button :label="1"><%=rb.getString("SheBeiZu")%></el-radio-button>
								</el-radio-group>
						</el-form-item>
						
						<!-- 设备组列表 -->
						<el-form-item label-width="0" class="el-form-table-container" v-show="ruleForm.enbDeviceByDeviceGroup == '1'" style="height:300px;">
							<el-ctable  @selection-change='enbDeviceGroupSelect' @load-success="tableLoadSuccess('enbDeviceGroup')" border=true :readonly='viewFlag || ruleForm.enbAllAndCustomize == 0' ref="deviceGroupTable" :default-checked="enbDeviceChecked" row-key="id" :rownumber="true" :page-size="pageSize" pagination="true" :page-list="pageList" :query-params="queryDeviceForm" :url="deviceGroupUrl" style="width:60%;height:100%;border:1px solid #E9E9E9;">
								<el-table-column type="selection" reserve-selection width=55></el-table-column>
								<el-table-column prop="group_name" label="<%=rb.getString("SheBeiZuMingCheng") %>"></el-table-column>
							</el-ctable>
						</el-form-item>
						
						<!-- 基站设备列表  -->
						<el-form-item label-width="0" v-show="ruleForm.enbDeviceByDeviceGroup != '1' " class="el-form-pagrid-container"   key="enbTable">
							<el-pairgrid @selection-change='enbSelectChange' @right-load-success="tableLoadSuccess('enbDevice')" ref="cpairgrid" query-name="serial_number" :rownumber="true" :page-list="pageList" :right-url="enbRightUrl" :left-url="enbLeftUrl" :height="height" :readonly='viewFlag' row-key="small_cell_code" :query-params="queryForm" :title="deviceTitle" :messages="{placeholder:'<%=rb.getString("XiaoZhanBianMa")%>'}">
								<template slot="prev">
									<el-ctable ref="groupFiltor" style="width:250px;height:100%;" :show-pager="false" row-key="id" :url="deviceGroupUrlAll" rownumber="true" :page-size="pageSize" pagination="false" :page-list="pageList" :query-params="queryDeviceForm"
										@row-click="groupRowClick">
										<template slot="toolbar">
											<span style="padding-left:10px;"><%=rb.getString("SheBeiZu") %></span>
										</template> 
										<el-table-column prop="group_name" label="<%=rb.getString("SheBeiZuMingCheng") %>"></el-table-column>
									</el-ctable>
								</template>
								<template slot="left">
									<el-table-column type="selection" width="45"></el-table-column>
									<el-table-column prop="connection_status" width="50">
										<template slot-scope="scope">
											<div :class="{
												'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
												'':scope.row.have_connected==2,
												'conn_exc':scope.row.connection_status=='Exception',
												'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
										</template>
									</el-table-column>
									<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
									<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>'></el-table-column>
								</template>
								<template slot='toolbar'>
									<el-form :model='queryForm' ref="queryForm" label-position="top">
										<div style="flex:1 1 auto;max-width:300px;border-radius:2px;border:1px solid #DEDFE6;display:flex;align-items:center;padding-right:12px;margin-left:20px;">
											<el-input class='pairgrid-query' style='width:400px;padding-left:10px;' v-model="queryForm.searchText" @keyup.enter.native="query"
												placeholder="<%=rb.getString("XiaoZhanBianMa")%>" size="mini" ></el-input>
											<el-input style="display:none"></el-input>
											<i @click='query' class="el-icon el-icon-common-search"></i>
										</div>
									</el-form>
								</template>
								<template slot='right'>
									<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
									<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>'></el-table-column>
								</template>
							</el-pairgrid>
						</el-form-item>
					</div>
				</el-tab-pane>
				<el-tab-pane label="UPS" name="UPS"  v-if="deviceTypeShowUPS == true">
					<div class="concentBoxUPS">
						<!-- UPS设备选择 --><!-- <%=rb.getString("SheBeiXuanZe")%> -->
						<el-form-item  prop='upsDeviceByDeviceGroup' label='' label-width="0px" :class="ruleForm.upsDeviceByDeviceGroup == 1 ? 'radioButtonSelect' : ''"> 
								<el-radio-group v-model='ruleForm.upsAllAndCustomize' :disabled='viewFlag' style="padding-top:17px;" @change="upsAllAndCustomizeChange">
									<el-radio :label="0" ><%=rb.getString("QuanXuan")%></el-radio>
									<el-radio :label="1" style="margin-left:50px"><%=rb.getString("DingZhi")%></el-radio>
								</el-radio-group>
								<el-radio-group v-model="ruleForm.upsDeviceByDeviceGroup" @change="upsDeviceByDeviceGroupChange" :disabled='ruleForm.upsAllAndCustomize == 0 || viewFlag' size="mini" style="padding-top:15px;padding-left:5px;">
										<el-radio-button :label="0"><%=rb.getString("KPISheBei")%></el-radio-button>
										<el-radio-button :label="1"><%=rb.getString("SheBeiZu")%></el-radio-button>
								</el-radio-group>
						</el-form-item>
						
						<!-- ups设备组列表 -->
						<el-form-item label-width="0" class="el-form-table-container" v-show="ruleForm.upsDeviceByDeviceGroup == '1' " style="height:300px;">
							<el-ctable  @selection-change='upsDeviceGroupSelect' @load-success="tableLoadSuccess('upsDeviceGroup')" border=true :readonly='viewFlag || ruleForm.upsAllAndCustomize == 0' ref="upsDeviceGroupTable" :default-checked="upsDeviceChecked" row-key="id" :rownumber="true" :page-size="pageSize" pagination="true" :page-list="pageList" :query-params="queryDeviceForm" :url="upsDeviceGroupUrl" style="width:60%;height:100%;border:1px solid #E9E9E9;">
								<el-table-column type="selection" reserve-selection width=55></el-table-column>
								<el-table-column prop="group_name" label="<%=rb.getString("SheBeiZuMingCheng") %>"></el-table-column>
							</el-ctable>
						</el-form-item>
						
						<!-- ups设备列表  -->
						<el-form-item label-width="0" v-show="ruleForm.upsDeviceByDeviceGroup != '1'" class="el-form-pagrid-container"   key="upsTable">
							<el-pairgrid @selection-change='upsSelectChange' @right-load-success="tableLoadSuccess('upsDevice')" ref="cpairgridUPS" query-name="serial_number" :rownumber="true" :page-list="pageList" :right-url="upsRightUrl" :left-url="upsLeftUrl" :height="height" :readonly='viewFlag' row-key="serial_number" :query-params="queryFormUPS" :title="deviceTitle" :messages="{placeholder:'<%=rb.getString("DianYuanBianMa")%>'}">
								<template slot="prev">
									<el-ctable ref="upsGroupFiltor" style="width:250px;height:100%;" :show-pager="false" row-key="id" :url="upsDeviceGroupUrlAll" rownumber="true" :page-size="pageSize" pagination="false" :page-list="pageList" :query-params="queryDeviceForm"
										@row-click="upsGroupRowClick">
										<template slot="toolbar">
											<span style="padding-left:10px;"><%=rb.getString("SheBeiZu") %></span>
										</template> 
										<el-table-column prop="group_name" label="<%=rb.getString("SheBeiZuMingCheng") %>"></el-table-column>
									</el-ctable>
								</template>
								<template slot="left">
									<el-table-column type="selection" width="45"></el-table-column>
									<el-table-column prop="connection_status" width="50">
										<template slot-scope="scope">
											<div :class="{
												'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
												'':scope.row.have_connected==2,
												'conn_exc':scope.row.connection_status=='Exception',
												'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
										</template>
									</el-table-column>
									<el-table-column prop='serial_number' label='<%=rb.getString("DianYuanBianMa")%>'></el-table-column>
									<el-table-column prop='device_name' label='<%=rb.getString("Title_SheBeiMingCheng")%>'></el-table-column>
								</template>
								<template slot='toolbar'>
									<el-form :model='queryFormUPS' ref="queryFormUPS" label-position="top">
										<div style="flex:1 1 auto;max-width:300px;border-radius:2px;border:1px solid #DEDFE6;display:flex;align-items:center;padding-right:12px;margin-left:20px;">
											<el-input class='pairgrid-query' style='width:400px;padding-left:10px;' v-model="queryFormUPS.searchText" @keyup.enter.native="queryUPS"
												placeholder="<%=rb.getString("DianYuanBianMa")%>" size="mini" ></el-input>
											<el-input style="display:none"></el-input>
											<i @click='queryUPS' class="el-icon el-icon-common-search"></i>
										</div>
									</el-form>
								</template>
								<template slot='right'>
									<el-table-column prop='serial_number' label='<%=rb.getString("DianYuanBianMa")%>'></el-table-column>
									<el-table-column prop='device_name' label='<%=rb.getString("Title_SheBeiMingCheng")%>'></el-table-column>
								</template>
							</el-pairgrid>
						</el-form-item>
					</div>
				</el-tab-pane>
				<el-tab-pane label="CPE" name="CPE"  v-if="deviceTypeShowCPE == true">
					<div class="concentBoxCPE">
						<!-- CPE设备选择 --><!-- <%=rb.getString("SheBeiXuanZe")%> -->
						<el-form-item  prop='cpeDeviceByDeviceGroup' label='' label-width="0px" :class="ruleForm.cpeDeviceByDeviceGroup == 1 ? 'radioButtonSelect' : ''"> 
								<el-radio-group v-model='ruleForm.cpeAllAndCustomize' :disabled='viewFlag' style="padding-top:17px;" @change="cpeAllAndCustomizeChange">
									<el-radio :label="0" ><%=rb.getString("QuanXuan")%></el-radio>
									<el-radio :label="1" style="margin-left:50px"><%=rb.getString("DingZhi")%></el-radio>
								</el-radio-group>
								<el-radio-group v-model="ruleForm.cpeDeviceByDeviceGroup" @change="cpeDeviceByDeviceGroupChange" :disabled='ruleForm.cpeAllAndCustomize == 0 || viewFlag' size="mini" style="padding-top:15px;padding-left:5px;">
										<el-radio-button :label="0"><%=rb.getString("KPISheBei")%></el-radio-button>
										<el-radio-button :label="1"><%=rb.getString("SheBeiZu")%></el-radio-button>
								</el-radio-group>
						</el-form-item>
						
						<!-- 设备组列表 -->
						<el-form-item label-width="0" class="el-form-table-container" v-show="ruleForm.cpeDeviceByDeviceGroup == '1'" style="height:300px;">
							<el-ctable  @selection-change='cpeDeviceGroupSelect' @load-success="tableLoadSuccess('cpeDeviceGroup')" border=true :readonly='viewFlag || ruleForm.cpeAllAndCustomize == 0' ref="cpeDeviceGroupTable" :default-checked="cpeDeviceChecked" row-key="id" :rownumber="true" :page-size="pageSize" pagination="true" :page-list="pageList" :query-params="queryDeviceForm" :url="cpeDeviceGroupUrl" style="width:60%;height:100%;border:1px solid #E9E9E9;">
								<el-table-column type="selection" reserve-selection width=55></el-table-column>
								<el-table-column prop="group_name" label="<%=rb.getString("SheBeiZuMingCheng") %>"></el-table-column>
							</el-ctable>
						</el-form-item>
						
						<!-- CPE设备列表  -->
						<el-form-item label-width="0" v-show="ruleForm.cpeDeviceByDeviceGroup != '1' " class="el-form-pagrid-container"    key="cpeTable">
							<el-pairgrid @selection-change='cpeSelectChange' @right-load-success="tableLoadSuccess('cpeDevice')" ref="cpairgridCPE" query-name="macaddress" :rownumber="true" :page-list="pageList" :right-url="cpeRightUrl" :left-url="cpeLeftUrl" :height="height" :readonly='viewFlag' row-key="cpe_code" :query-params="queryFormCPE" :title="deviceTitle" :messages="{placeholder:'<%=rb.getString("MACDiZhi")%>'}">
								<template slot="prev">
									<el-ctable ref="cpeGroupFiltor" style="width:250px;height:100%;" :show-pager="false" row-key="id" :url="cpeDeviceGroupUrlAll" rownumber="true" :page-size="pageSize" pagination="false" :page-list="pageList" :query-params="queryDeviceForm"
										@row-click="cpeGroupRowClick">
										<template slot="toolbar">
											<span style="padding-left:10px;"><%=rb.getString("SheBeiZu") %></span>
										</template> 
										<el-table-column prop="group_name" label="<%=rb.getString("SheBeiZuMingCheng") %>"></el-table-column>
									</el-ctable>
								</template>
								<template slot="left">
									<el-table-column type="selection" width="45"></el-table-column>
									<el-table-column prop="connection_status" width="50">
										<template slot-scope="scope">
											<div :class="{
												'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
												'':scope.row.have_connected==2,
												'conn_exc':scope.row.connection_status=='Exception',
												'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
										</template>
									</el-table-column>
									<el-table-column prop='serial_number' label='<%=rb.getString("CPEXuLieHao")%>'></el-table-column>
									<el-table-column prop='macaddress' label='<%=rb.getString("MACDiZhi")%>'></el-table-column>
									<el-table-column prop='cpe_name' label='<%=rb.getString("CPEName")%>'></el-table-column>
								</template>
								<template slot='toolbar'>
									<el-form :model='queryFormCPE' ref="queryFormCPE" label-position="top">
										<div style="flex:1 1 auto;max-width:300px;border-radius:2px;border:1px solid #DEDFE6;display:flex;align-items:center;padding-right:12px;margin-left:20px;">
											<el-input class='pairgrid-query' style='width:400px;padding-left:10px;' v-model="queryFormCPE.searchText" @keyup.enter.native="queryCPE"
												placeholder="<%=rb.getString("CPEXuLieHao")%>" size="mini" ></el-input>
											<el-input style="display:none"></el-input>
											<i @click='queryCPE' class="el-icon el-icon-common-search"></i>
										</div>
									</el-form>
								</template>
								<template slot='right'>
									<el-table-column prop='serial_number' label='<%=rb.getString("CPEXuLieHao")%>'></el-table-column>
									<el-table-column prop='cpe_name' label='<%=rb.getString("CPEName")%>'></el-table-column>
									<el-table-column prop='macaddress' label='<%=rb.getString("MACDiZhi")%>' min-width="200"></el-table-column>
								</template>
							</el-pairgrid>
						</el-form-item>
					</div>
				</el-tab-pane>
				<el-tab-pane label="GNB" name="GNB"  v-if="deviceTypeShowGNB == true">
					<div class="concentBoxCPE">
						<!-- GNB设备选择 --><!-- <%=rb.getString("SheBeiXuanZe")%> -->
						<el-form-item  prop='gnbDeviceByDeviceGroup' label='' label-width="0px" :class="ruleForm.gnbDeviceByDeviceGroup == 1 ? 'radioButtonSelect' : ''"> 
								<el-radio-group v-model='ruleForm.gnbAllAndCustomize' :disabled='viewFlag' style="padding-top:17px;" @change="gnbAllAndCustomizeChange">
									<el-radio :label="0" ><%=rb.getString("QuanXuan")%></el-radio>
									<el-radio :label="1" style="margin-left:50px"><%=rb.getString("DingZhi")%></el-radio>
								</el-radio-group>
								<el-radio-group v-model="ruleForm.gnbDeviceByDeviceGroup" @change="gnbDeviceByDeviceGroupChange" :disabled='ruleForm.gnbAllAndCustomize == 0 || viewFlag' size="mini" style="padding-top:15px;padding-left:5px;">
										<el-radio-button :label="0"><%=rb.getString("KPISheBei")%></el-radio-button>
										<el-radio-button :label="1"><%=rb.getString("SheBeiZu")%></el-radio-button>
								</el-radio-group>
						</el-form-item>
						
						<!-- 设备组列表 -->
						<el-form-item label-width="0" class="el-form-table-container" v-show="ruleForm.gnbDeviceByDeviceGroup == '1'" style="height:300px;">
							<el-ctable  @selection-change='gnbDeviceGroupSelect' @load-success="tableLoadSuccess('gnbDeviceGroup')" border=true :readonly='viewFlag || ruleForm.gnbAllAndCustomize == 0' ref="gnbDeviceGroupTable" :default-checked="gnbDeviceChecked" row-key="id" :rownumber="true" :page-size="pageSize" pagination="true" :page-list="pageList" :query-params="queryDeviceForm" :url="gnbDeviceGroupUrl" style="width:60%;height:100%;border:1px solid #E9E9E9;">
								<el-table-column type="selection" reserve-selection width=55></el-table-column>
								<el-table-column prop="group_name" label="<%=rb.getString("SheBeiZuMingCheng") %>"></el-table-column>
							</el-ctable>
						</el-form-item>
						
						<!-- GNB设备列表  -->
						<el-form-item label-width="0" v-show="ruleForm.gnbDeviceByDeviceGroup != '1' " class="el-form-pagrid-container"    key="gnbTable">
							<el-pairgrid @selection-change='gnbSelectChange' @right-load-success="tableLoadSuccess('gnbDevice')" ref="cpairgridGNB" query-name="serial_number" :rownumber="true" :page-list="pageList" :right-url="gnbRightUrl" :left-url="gnbLeftUrl" :height="height" :readonly='viewFlag' row-key="small_cell_code" :query-params="queryFormGNB" :title="deviceTitle" :messages="{placeholder:'<%=rb.getString("Title_SheBeiBianMa")%>'}">
								<template slot="prev">
									<el-ctable ref="gnbGroupFiltor" style="width:250px;height:100%;" :show-pager="false" row-key="id" :url="gnbDeviceGroupUrlAll" rownumber="true" :page-size="pageSize" pagination="false" :page-list="pageList" :query-params="queryDeviceForm"
										@row-click="gnbGroupRowClick">
										<template slot="toolbar">
											<span style="padding-left:10px;"><%=rb.getString("SheBeiZu") %></span>
										</template> 
										<el-table-column prop="group_name" label="<%=rb.getString("SheBeiZuMingCheng") %>"></el-table-column>
									</el-ctable>
								</template>
								<template slot="left">
									<el-table-column type="selection" width="45"></el-table-column>
									<el-table-column prop="connection_status" width="50">
										<template slot-scope="scope">
											<div :class="{
												'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
												'':scope.row.have_connected==2,
												'conn_exc':scope.row.connection_status=='Exception',
												'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
										</template>
									</el-table-column>
									<el-table-column prop='serial_number' label='<%=rb.getString("Title_SheBeiBianMa")%>'></el-table-column>
									<el-table-column prop='host_name' label='<%=rb.getString("Title_SheBeiMingCheng")%>'></el-table-column>
								</template>
								<template slot='toolbar'>
									<el-form :model='queryFormGNB' ref="queryFormGNB" label-position="top">
										<div style="flex:1 1 auto;max-width:300px;border-radius:2px;border:1px solid #DEDFE6;display:flex;align-items:center;padding-right:12px;margin-left:20px;">
											<el-input class='pairgrid-query' style='width:400px;padding-left:10px;' v-model="queryFormGNB.searchText" @keyup.enter.native="queryGNB"
												placeholder="<%=rb.getString("Title_SheBeiBianMa")%>" size="mini" ></el-input>
											<el-input style="display:none"></el-input>
											<i @click='queryGNB' class="el-icon el-icon-common-search"></i>
										</div>
									</el-form>
								</template>
								<template slot='right'>
									<el-table-column prop='serial_number' label='<%=rb.getString("Title_SheBeiBianMa")%>'></el-table-column>
									<el-table-column prop='host_name' label='<%=rb.getString("Title_SheBeiMingCheng")%>'></el-table-column>
								</template>
							</el-pairgrid>
						</el-form-item>
					</div>
				</el-tab-pane>
				<el-tab-pane label="WCG" name="EGW"  v-if="deviceTypeShowEGW == true">
					<div class="concentBoxCPE">
						<!-- EGW设备选择 --><!-- <%=rb.getString("SheBeiXuanZe")%> -->
						<el-form-item  prop='egwDeviceByDeviceGroup' label='' label-width="0px" :class="ruleForm.egwDeviceByDeviceGroup == 1 ? 'radioButtonSelect' : ''"> 
								<el-radio-group v-model='ruleForm.egwAllAndCustomize' :disabled='viewFlag' style="padding-top:17px;" @change="egwAllAndCustomizeChange">
									<el-radio :label="0" ><%=rb.getString("QuanXuan")%></el-radio>
									<el-radio :label="1" style="margin-left:50px"><%=rb.getString("DingZhi")%></el-radio>
								</el-radio-group>
								<el-radio-group v-model="ruleForm.egwDeviceByDeviceGroup" @change="egwDeviceByDeviceGroupChange" :disabled='ruleForm.egwAllAndCustomize == 0 || viewFlag' size="mini" style="padding-top:15px;padding-left:5px;">
										<el-radio-button :label="0"><%=rb.getString("KPISheBei")%></el-radio-button>
										<el-radio-button :label="1"><%=rb.getString("SheBeiZu")%></el-radio-button>
								</el-radio-group>
						</el-form-item>
						
						<!-- 设备组列表 -->
						<el-form-item label-width="0" class="el-form-table-container" v-show="ruleForm.egwDeviceByDeviceGroup == '1'" style="height:300px;">
							<el-ctable key="gpet"
								@selection-change='egwDeviceGroupSelect'
								@load-success="tableLoadSuccess('egwDeviceGroup')" :border="true"
								:readonly='viewFlag || ruleForm.egwAllAndCustomize == 0' ref="egwDeviceGroupTable" 
								:default-checked="egwDeviceChecked"
								row-key="id" 
								:query-params="queryDeviceForm" :url="egwDeviceGroupUrl" style="width:60%;height:100%;border:1px solid #E9E9E9;">
								<el-table-column type="selection" reserve-selection width="55" key="egws"></el-table-column>
								<el-table-column prop="group_name" label="<%=rb.getString("SheBeiZuMingCheng") %>"></el-table-column>
							</el-ctable>
						</el-form-item>
						
						<!-- EGW设备列表  -->
						<el-form-item label-width="0" v-show="ruleForm.egwDeviceByDeviceGroup != '1' " class="el-form-pagrid-container"  key="egwTable">
							<el-pairgrid @selection-change='egwSelectChange' @right-load-success="tableLoadSuccess('egwDevice')" ref="cpairgridEGW" query-name="serial_number" :rownumber="true" :page-list="pageList" :right-url="egwRightUrl" :left-url="egwLeftUrl" :height="height" :readonly='viewFlag' row-key="egw_code" :query-params="queryFormEGW" :title="deviceTitle" :messages="{placeholder:'<%=rb.getString("eGWBianMa")%>'}">
								<template slot="prev">
									<el-ctable ref="egwGroupFiltor" style="width:250px;height:100%;" :show-pager="false" row-key="id" :url="egwDeviceGroupUrlAll" rownumber="true" :page-size="pageSize" pagination="false" :page-list="pageList" :query-params="queryDeviceForm"
										@row-click="egwGroupRowClick">
										<template slot="toolbar">
											<span style="padding-left:10px;"><%=rb.getString("SheBeiZu") %></span>
										</template> 
										<el-table-column prop="group_name" label="<%=rb.getString("SheBeiZuMingCheng") %>"></el-table-column>
									</el-ctable>
								</template>
								<template slot="left">
									<el-table-column type="selection" width="45"></el-table-column>
									<el-table-column prop="connection_status" width="50">
										<template slot-scope="scope">
											<div :class="{
												'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
												'':scope.row.have_connected==2,
												'conn_exc':scope.row.connection_status=='Exception',
												'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
										</template>
									</el-table-column>
									<el-table-column prop='serial_number' label='<%=rb.getString("eGWBianMa")%>'></el-table-column>
									<el-table-column prop='host_name' label='<%=rb.getString("eGWMingCheng")%>'></el-table-column>
								</template>
								<template slot='toolbar'>
									<el-form :model='queryFormEGW' ref="queryFormEGW" label-position="top">
										<div style="flex:1 1 auto;max-width:300px;border-radius:2px;border:1px solid #DEDFE6;display:flex;align-items:center;padding-right:12px;margin-left:20px;">
											<el-input class='pairgrid-query' style='width:400px;padding-left:10px;' v-model="queryFormEGW.searchText" @keyup.enter.native="queryEGW"
												placeholder="<%=rb.getString("eGWBianMa")%>" size="mini" ></el-input>
											<el-input style="display:none"></el-input>
											<i @click='queryEGW' class="el-icon el-icon-common-search"></i>
										</div>
									</el-form>
								</template>
								<template slot='right'>
									<el-table-column prop='serial_number' label='<%=rb.getString("eGWBianMa")%>'></el-table-column>
									<el-table-column prop='host_name' label='<%=rb.getString("eGWMingCheng")%>'></el-table-column>
								</template>
							</el-pairgrid>
						</el-form-item>
					</div>
				</el-tab-pane>
				<el-tab-pane label="GSM" name="GSM"  v-if="deviceTypeShowGSM == true">
					<div class="concentBoxGSM">
						<!-- GSM设备选择 -->
						<el-form-item  prop='gsmDeviceByDeviceGroup' label='' label-width="0px" :class="ruleForm.gsmDeviceByDeviceGroup == 1 ? 'radioButtonSelect' : ''"> 
								<el-radio-group v-model='ruleForm.gsmAllAndCustomize' :disabled='viewFlag' style="padding-top:17px;" @change="gsmAllAndCustomizeChange">
									<el-radio :label="0" ><%=rb.getString("QuanXuan")%></el-radio>
									<el-radio :label="1" style="margin-left:50px"><%=rb.getString("DingZhi")%></el-radio>
								</el-radio-group>
								<el-radio-group v-model="ruleForm.gsmDeviceByDeviceGroup" @change="gsmDeviceByDeviceGroupChange"  :disabled='ruleForm.gsmAllAndCustomize == 0 || viewFlag' size="mini" style="padding-top:15px;padding-left:5px;">
										<el-radio-button :label="0"><%=rb.getString("KPISheBei")%></el-radio-button>
										<el-radio-button :label="1"><%=rb.getString("SheBeiZu")%></el-radio-button>
								</el-radio-group>
						</el-form-item>
						<!-- 设备组列表 -->
						<el-form-item label-width="0" class="el-form-table-container" v-show="ruleForm.gsmDeviceByDeviceGroup == '1'" style="height:300px;">
							<el-ctable  @selection-change='gsmDeviceGroupSelect' @load-success="tableLoadSuccess('gsmDeviceGroup')" border=true :readonly='viewFlag || ruleForm.gsmAllAndCustomize == 0' ref="gsmDeviceGroupTable" :default-checked="gsmDeviceChecked" row-key="id" :rownumber="true" :page-size="pageSize" pagination="true" :page-list="pageList" :query-params="queryDeviceForm" :url="gsmDeviceGroupUrl" style="width:60%;height:100%;border:1px solid #E9E9E9;">
								<el-table-column type="selection" reserve-selection width=55></el-table-column>
								<el-table-column prop="group_name" label="<%=rb.getString("SheBeiZuMingCheng") %>"></el-table-column>
							</el-ctable>
						</el-form-item>
						<!-- 基站设备列表  -->
						<el-form-item label-width="0" v-show="ruleForm.gsmDeviceByDeviceGroup != '1' " class="el-form-pagrid-container"   key="gsmTable">
							<el-pairgrid @selection-change='gsmSelectChange' @right-load-success="tableLoadSuccess('gsmDevice')" ref="cpairgridGSM" query-name="serial_number" :rownumber="true" :page-list="pageList" :right-url="gsmRightUrl" :left-url="gsmLeftUrl" :height="height" :readonly='viewFlag' row-key="small_cell_code" :query-params="queryFormGSM" :title="deviceTitle" :messages="{placeholder:'<%=rb.getString("XiaoZhanBianMa")%>'}">
								<template slot="prev">
									<el-ctable ref="gsmGroupFiltor" style="width:250px;height:100%;" :show-pager="false" row-key="id" :url="gsmDeviceGroupUrlAll" rownumber="true" :page-size="pageSize" pagination="false" :page-list="pageList" :query-params="queryDeviceForm"
										@row-click="gsmGroupRowClick">
										<template slot="toolbar">
											<span style="padding-left:10px;"><%=rb.getString("SheBeiZu") %></span>
										</template> 
										<el-table-column prop="group_name" label="<%=rb.getString("SheBeiZuMingCheng") %>"></el-table-column>
									</el-ctable>
								</template>
								<template slot="left">
									<el-table-column type="selection" width="45"></el-table-column>
									<el-table-column prop="connection_status" width="50">
										<template slot-scope="scope">
											<div :class="{
												'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
												'':scope.row.have_connected==2,
												'conn_exc':scope.row.connection_status=='Exception',
												'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
										</template>
									</el-table-column>
									<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
									<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>'></el-table-column>
								</template>
								<template slot='toolbar'>
									<el-form :model='queryFormGSM' ref="queryFormGSM" label-position="top">
										<div style="flex:1 1 auto;max-width:300px;border-radius:2px;border:1px solid #DEDFE6;display:flex;align-items:center;padding-right:12px;margin-left:20px;">
											<el-input class='pairgrid-query' style='width:400px;padding-left:10px;' v-model="queryFormGSM.searchText" @keyup.enter.native="queryGSM"
												placeholder="<%=rb.getString("XiaoZhanBianMa")%>" size="mini" ></el-input>
											<el-input style="display:none"></el-input>
											<i @click='queryGSM' class="el-icon el-icon-common-search"></i>
										</div>
									</el-form>
								</template>
								<template slot='right'>
									<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
									<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>'></el-table-column>
								</template>
							</el-pairgrid>
						</el-form-item>
					</div>
				</el-tab-pane>
			</el-tabs>
			<!-- 告警列表  -->
			<el-form-item label-width="0" class="el-form-pagrid-container" style="margin-left:34px;position:relative;">
				<div class="allAlarmClass">
					<el-checkbox v-model="ruleForm.allAlarm" :true-label="1" :false-label="0" :disabled='viewFlag' @change="allAlarmChange"><%=rb.getString("QuanXuan")%></el-checkbox>
				</div>
				<div v-show="ruleForm.allAlarm == 1" style="font-size:14px;"><%=rb.getString("XuanZeGaoJing")%></div>
				<el-pairgrid v-if="ruleForm.allAlarm == 0" @checked-change='alarmSelectChange' @right-load-success="tableLoadSuccess('alarmList')" :readonly='viewFlag' ref="calarmPairgrid" query-name="ALARM_IDENTIFIER" :rownumber="true" :page-list="pageList" :right-url="alarmRightUrl" :left-url="alarmLeftUrl" :height="height" row-key="ALARM_IDENTIFIER" :query-params="alarmQueryForm" :title="alarmDeviceTitle" :messages="{placeholder:'<%=rb.getString("GaoJingWeiYiBiaoZhi")%>'}">
					<template slot="left">
						<el-table-column type="selection" width="45"></el-table-column>
						<el-table-column prop='DEVICE_TYPE_NAME' label='<%=rb.getString("XinGaoJingYuan")%>'></el-table-column>
						<el-table-column prop='ALARM_IDENTIFIER' label='<%=rb.getString("GaoJingWeiYiBiaoZhi")%>' ></el-table-column>
						<el-table-column prop='ALARM_NAME' label='<%=rb.getString("KeNengYuanYin")%>' width=""></el-table-column>
						<el-table-column label='<%=rb.getString("ShiJianLeiXing")%>' min-width="150" prop="EVENT_TYPE" show-overflow-tooltip>
							<template slot-scope="scope">
								<div v-if="scope.row.EVENT_TYPE == '30000'">
									<span><%=rb.getString("TongXinGaoJing")%></span>
								</div>
								<div v-else-if="scope.row.EVENT_TYPE == '30001'">
									<span><%=rb.getString("FuWuZhiLiangGaoJing")%></span>
								</div>
								<div v-if="scope.row.EVENT_TYPE == '30002'">
									<span><%=rb.getString("ChuLiShiBaiGaoJing")%></span>
								</div>
								<div v-else-if="scope.row.EVENT_TYPE == '30003'">
									<span><%=rb.getString("SheBeiGaoJing")%></span>
								</div>
								<div v-else-if="scope.row.EVENT_TYPE == '30004'">
									<span><%=rb.getString("HuanJingGaoJing")%></span>
								</div>
								<div v-else-if="scope.row.EVENT_TYPE == '30006'">
									<span><%=rb.getString("XingNengYiChuGaoJing")%></span>
								</div>
							</template>
						</el-table-column>
						<el-table-column prop='SERVERITY_TYPE' label='<%=rb.getString("GaoJingJiBie")%>' >
							<template slot-scope="scope">
								<div v-if="scope.row.SERVERITY_TYPE == 'Minor'" class="alarmMinor">
									<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("CiYaoGaoJing")%>
								</div>
								<div v-else-if="scope.row.SERVERITY_TYPE == 'Major'"  class="alarmMajor">
									<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("ZhuYaoGaoJing")%>
								</div>
								<div v-else-if="scope.row.SERVERITY_TYPE == 'Critical'" class="alarmCritical">
									<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("JinJiGaoJing")%>
								</div>
								<div v-else-if="scope.row.SERVERITY_TYPE == 'Warning'" class="alarmWarning">
									<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("JingGaoGaoJing")%>
								</div>
								
							</template>
						</el-table-column>
					</template>
					<template slot='toolbar'>
						<div style="display: flex;align-items: center;">
							<el-query type="normal" @query="queryAlarm"  placeholder="<%=rb.getString("GaoJingWeiYiBiaoZhi")%>/<%=rb.getString("KeNengYuanYin")%>"></el-query>
							<el-popfilter style="margin: 0 5px;"
								type="single"
								label='<%=rb.getString("ShiJianLeiXing")%>'
								v-model="alarmQueryFormData.alarmType"
								:list="eventTypeList.map(item=>{return {label:item.text,value:item.id}})"
								@check-change="alarmQueryAdvance">
							</el-popfilter>
							<el-popfilter style="margin: 0 5px;"
								type="single"
								label='<%=rb.getString("YanZhongChengDu")%>'
								v-model="alarmQueryFormData.alarmServerity"
								:list="alarmSeverityList.map(item=>{return {label:item.text,value:item.id}})"
								@check-change="alarmQueryAdvance">
							</el-popfilter>
							<el-popfilter style="margin: 0 5px;"
								type="single"
								label='<%=rb.getString("XinGaoJingYuan")%>'
								v-model="alarmQueryFormData.alarmSource"
								:list="alarmSourceList.map(item=>{return {label:item.label,value:item.value}})"
								@check-change="alarmQueryAdvance">
							</el-popfilter>
							<div class="pop-filter-clear" style="margin: 0 5px;" 
								@click="alarmReset">
								<%=rb.getString("QingKongShaiXuan")%>
							</div>
						</div>
					</template>
					<template slot='right'>
						<el-table-column prop='ALARM_IDENTIFIER' label='<%=rb.getString("GaoJingWeiYiBiaoZhi")%>'></el-table-column>
						<el-table-column prop='SERVERITY_TYPE' label='<%=rb.getString("GaoJingJiBie")%>'>
							<template slot-scope="scope">
								<div v-if="scope.row.SERVERITY_TYPE == 'Minor'" class="alarmMinor">
									<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("CiYaoGaoJing")%>
								</div>
								<div v-else-if="scope.row.SERVERITY_TYPE == 'Major'"  class="alarmMajor">
									<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("ZhuYaoGaoJing")%>
								</div>
								<div v-else-if="scope.row.SERVERITY_TYPE == 'Critical'" class="alarmCritical">
									<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("JinJiGaoJing")%>
								</div>
								<div v-else-if="scope.row.SERVERITY_TYPE == 'Warning'" class="alarmWarning">
									<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("JingGaoGaoJing")%>
								</div>
							</template>
						</el-table-column>
					</template>
				</el-pairgrid>
				<el-ctable style="border:1px solid #e1e1e1;" v-if="ruleForm.allAlarm == 1"  @load-success="tableLoadSuccess('alarmList')" :readonly='viewFlag || ruleForm.allAlarm == 1' ref="calarmPairgrids" query-name="ALARM_IDENTIFIER" :rownumber="true" :page-size="pageSize" :page-list="pageList"  :url="alarmLeftUrl" :height="height" row-key="ALARM_IDENTIFIER" :query-params="alarmQueryFormAll" :title="alarmDeviceTitle" :messages="{placeholder:'<%=rb.getString("GaoJingWeiYiBiaoZhi")%>'}">
					<template slot='toolbar'>
						<div style="display: flex;align-items: center;">
							<el-query type="normal" @query="queryAlarmAll"  placeholder="<%=rb.getString("GaoJingWeiYiBiaoZhi")%>/<%=rb.getString("KeNengYuanYin")%>"></el-query>
							<el-popfilter style="margin: 0 5px;"
								type="single"
								label='<%=rb.getString("ShiJianLeiXing")%>'
								v-model="alarmQueryFormDataAll.alarmType"
								:list="eventTypeList.map(item=>{return {label:item.text,value:item.id}})"
								@check-change="alarmQueryAdvanceAll">
							</el-popfilter>
							<el-popfilter style="margin: 0 5px;"
								type="single"
								label='<%=rb.getString("YanZhongChengDu")%>'
								v-model="alarmQueryFormDataAll.alarmServerity"
								:list="alarmSeverityList.map(item=>{return {label:item.text,value:item.id}})"
								@check-change="alarmQueryAdvanceAll">
							</el-popfilter>
							<el-popfilter style="margin: 0 5px;"
								type="single"
								label='<%=rb.getString("XinGaoJingYuan")%>'
								v-model="alarmQueryFormDataAll.alarmSource"
								:list="alarmSourceList.map(item=>{return {label:item.label,value:item.value}})"
								@check-change="alarmQueryAdvanceAll">
							</el-popfilter>
							<div class="pop-filter-clear" style="margin: 0 5px;" 
								@click="alarmResetAll">
								<%=rb.getString("QingKongShaiXuan")%>
							</div>
						</div>
					</template>
					<el-table-column type="selection" width="45"></el-table-column>
					<el-table-column prop='DEVICE_TYPE_NAME' label='<%=rb.getString("XinGaoJingYuan")%>'></el-table-column>
					<el-table-column prop='ALARM_IDENTIFIER' label='<%=rb.getString("GaoJingWeiYiBiaoZhi")%>' ></el-table-column>
					<el-table-column prop='ALARM_NAME' label='<%=rb.getString("KeNengYuanYin")%>' width=""></el-table-column>
					<el-table-column label='<%=rb.getString("ShiJianLeiXing")%>'  prop="EVENT_TYPE" show-overflow-tooltip>
						<template slot-scope="scope">
							<div v-if="scope.row.EVENT_TYPE == '30000'">
								<span><%=rb.getString("TongXinGaoJing")%></span>
							</div>
							<div v-else-if="scope.row.EVENT_TYPE == '30001'">
								<span><%=rb.getString("FuWuZhiLiangGaoJing")%></span>
							</div>
							<div v-if="scope.row.EVENT_TYPE == '30002'">
								<span><%=rb.getString("ChuLiShiBaiGaoJing")%></span>
							</div>
							<div v-else-if="scope.row.EVENT_TYPE == '30003'">
								<span><%=rb.getString("SheBeiGaoJing")%></span>
							</div>
							<div v-else-if="scope.row.EVENT_TYPE == '30004'">
								<span><%=rb.getString("HuanJingGaoJing")%></span>
							</div>
							<div v-else-if="scope.row.EVENT_TYPE == '30006'">
								<span><%=rb.getString("XingNengYiChuGaoJing")%></span>
							</div>
						</template>
					</el-table-column>
					<el-table-column prop='SERVERITY_TYPE' label='<%=rb.getString("GaoJingJiBie")%>' >
						<template slot-scope="scope">
							<div v-if="scope.row.SERVERITY_TYPE == 'Minor'" class="alarmMinor">
								<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("CiYaoGaoJing")%>
							</div>
							<div v-else-if="scope.row.SERVERITY_TYPE == 'Major'"  class="alarmMajor">
								<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("ZhuYaoGaoJing")%>
							</div>
							<div v-else-if="scope.row.SERVERITY_TYPE == 'Critical'" class="alarmCritical">
								<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("JinJiGaoJing")%>
							</div>
							<div v-else-if="scope.row.SERVERITY_TYPE == 'Warning'" class="alarmWarning">
								<span class="el-icon el-icon-status-alarm" style="margin-right:5px;"></span><%=rb.getString("JingGaoGaoJing")%>
							</div>
							
						</template>
					</el-table-column>
					
				</el-ctable>
			</el-form-item>
			<el-form-item prop='alarmId' style="margin-left:45px;margin-bottom:20px;" label-width="0" v-if="defaultVal !== '1'">
				<el-input v-model='ruleForm.alarmId' v-show="false"></el-input>
			</el-form-item>
			<el-form-item  prop="times"  label="<%=rb.getString("GuZhangShiJian")%>" style="margin-top:10px;" v-if="defaultVal !== '1'">
				<div style="padding-top:5px;">
					<el-date-picker
						v-model="ruleForm.startTime"
						type="datetime"
						:disabled='viewFlag'
						style="width:180px"
						value-format = "yyyy-MM-dd HH:mm:ss"
						placeholder="<%=rb.getString("KaiShiShiJian")%>">
					</el-date-picker>
					<span style="padding:0 10px;">-</span>
					<el-date-picker
						v-model="ruleForm.endTime"
						:disabled='viewFlag'
						style="width:180px"
						type="datetime"
						value-format = "yyyy-MM-dd HH:mm:ss"
						placeholder="<%=rb.getString("JieShuShiJian")%>">
					</el-date-picker>
				</div>
				
			</el-form-item>
		</div>
		
	</el-form>
	
</div>
<!-- 添加模板结束 -->
<script type="text/javascript">
if(window.filterTempalePage) {
	try {
		window.filterTempalePage.$destroy();
	}catch(e){}
}
window.filterTempalePage = new Vue({
	el:'#filterInfoPage',
	data(){
		var vm = this;
		var validateName = (rule,value,callback) => { //  规则标题验证
			if(value === ''){
				callback(new Error('<%=rb.getString("QingShuRuGuiZeMingCheng")%>'))
			}else if(value.length>50){
				callback(new Error('<%=rb.getString("ZiFuChang")%><%=rb.getString("MaoHao")%> 1~50<%=rb.getString("ZiFuFuShu")%>'))
			}else{
				axios.post('${ctx}/cell/fault/ruleNameExist.action',stringify({
					ruleName:vm.ruleForm.ruleName.trim(),
					ruleId:vm.ruleId
				})).then(function(response){
					var data = response.data;
					if(data["success"]){
						if(data["message"] == "true"){
							callback(new Error('<%=rb.getString("MingChengYiCunZai")%>'))
						}else{
							callback();
						}
					}
				}).catch(function(error){
					callback()
				})
			}
		};
		// 告警至少选择一个验证
		var validatorAlarmId = (rule,value,callback) => {
			if(this.ruleForm.allAlarm == 1){
				callback()
			}else{
				if(value == ''){
					callback(new Error('<%=rb.getString("QingXuanZeGaoJing")%>'))
				}else{
					callback()
				}
			}
			
		};
		return{
			defaultVal:'',
			initAlarmTypeArr: [],
			oldTemplateName:'',//校验名称是否已经存在时使用
			showFlag:true,
			enbDeviceSelection:[],
			upsDeviceSelection:[],
			cpeDeviceSelection:[],
			gnbDeviceSelection:[],
			egwDeviceSelection:[],
            gsmDeviceSelection:[],
			ruleId:'',
			actionType:'add',
			enbDeviceGroupSelection:[],
			upsDeviceGroupSelection:[],
			cpeDeviceGroupSelection:[],
			gnbDeviceGroupSelection:[],
			egwDeviceGroupSelection:[],
            gsmDeviceGroupSelection:[],
			viewFlag:false,
			enbDeviceChecked:[],
			upsDeviceChecked:[],
			cpeDeviceChecked:[],
			gnbDeviceChecked:[],
			egwDeviceChecked:[],
            gsmDeviceChecked:[],
			alarmTabsValue:'ENB',
			queryDeviceForm:{
				/* searchText:'', */
			},
			alarmDeviceTitle:['<%=rb.getString("XuanZeGaoJing")%>','<%=rb.getString("YiXuanGaoJing")%>'],
			alarmQueryForm:{
				search_text:'',
				alarmIdentifier:'',
				alarmType:'',
				alarmServerity:'',
				alarmSource:''
			},
			alarmQueryFormData:{
				alarmIdentifier:'',
				alarmType:'',
				alarmServerity:'',
				alarmSource:''
			},
			alarmQueryFormAll:{
				search_text:'',
				alarmIdentifier:'',
				alarmType:'',
				alarmServerity:'',
				alarmSource:''
			},
			alarmQueryFormDataAll:{
				alarmIdentifier:'',
				alarmType:'',
				alarmServerity:'',
				alarmSource:''
			},
			eventTypeList:[    //告警类型下拉选择数据 
						{text:'<%=rb.getString("QuanBu")%>',id:''},
						{text:'<%=rb.getString("TongXinGaoJing")%>',id:'30000'},
						{text:'<%=rb.getString("FuWuZhiLiangGaoJing")%>',id:'30001'},
						{text:'<%=rb.getString("ChuLiShiBaiGaoJing")%>',id:'30002'},
						{text:'<%=rb.getString("SheBeiGaoJing")%>',id:'30003'},
						{text:'<%=rb.getString("HuanJingGaoJing")%>',id:'30004'},
						// {text:'<%=rb.getString("XingNengYiChuGaoJing")%>',id:'30006'}
					],
			alarmSeverityList:[     // 告警级别下拉选择数据 
						{text:'<%=rb.getString("QuanBu")%>',id:''},
						{text:'<%=rb.getString("JinJiGaoJing")%>',id:'31001'},
						{text:'<%=rb.getString("ZhuYaoGaoJing")%>',id:'31002'},
						{text:'<%=rb.getString("CiYaoGaoJing")%>',id:'31003'},
						{text:'<%=rb.getString("JingGaoGaoJing")%>',id:'31004'}
					],
			queryForm:{
				searchText:'',
				deviceType:'ENB',
				groupId:''
			},
			queryFormUPS:{
				searchText:'',
				deviceType:'UPS',
				groupId:''
			},
			queryFormCPE:{
				searchText:'',
				deviceType:'CPE',
				groupId:''
			},
			queryFormGNB:{
				searchText:'',
				deviceType:'GNB',
				groupId:''
			},
			queryFormEGW:{
				searchText:'',
				deviceType:'EGW',
				groupId:''
			},
            queryFormGSM: {
				searchText:'',
				deviceType:'GSM',
				groupId:''
			},
			height:'370px',
			pageSize:50,
			pageList:[50,100,200],
			alarmLeftUrl:'${ctx}/cell/fault/queryAlarmLevelInfosList.action?deviceType=ENB&timeZone='+timeZone,
			alarmRightUrl:'',
			deviceGroupUrl:'',
			upsDeviceGroupUrl:'',
			cpeDeviceGroupUrl:'',
			gnbDeviceGroupUrl:'',
			egwDeviceGroupUrl:'',
            gsmDeviceGroupUrl:'',
			deviceGroupUrlAll:'${ctx}/cell/fault/queryDeviceGroupByOperPageList.action?deviceType=ENB',
			upsDeviceGroupUrlAll:'${ctx}/cell/fault/queryDeviceGroupByOperPageList.action?deviceType=UPS',
			cpeDeviceGroupUrlAll:'${ctx}/cell/fault/queryDeviceGroupByOperPageList.action?deviceType=CPE',
			gnbDeviceGroupUrlAll:'${ctx}/cell/fault/queryDeviceGroupByOperPageList.action?deviceType=GNB',
			egwDeviceGroupUrlAll:'${ctx}/cell/fault/queryDeviceGroupByOperPageList.action?deviceType=EGW',
            gsmDeviceGroupUrlAll:'${ctx}/cell/fault/queryDeviceGroupByOperPageList.action?deviceType=GSM',
			deviceTitle:['','<%=rb.getString("YiXuan")%>'], // <%=rb.getString("JiZhanLieBiao")%>
			enbLeftUrl:'${ctx}/cell/fault/queryDevicePageList.action',
			upsLeftUrl:'${ctx}/cell/fault/queryDevicePageList.action',
			cpeLeftUrl:'${ctx}/cell/fault/queryDevicePageList.action',
			gnbLeftUrl:'${ctx}/cell/fault/queryDevicePageList.action',
			egwLeftUrl:'${ctx}/cell/fault/queryDevicePageList.action',
            gsmLeftUrl:'${ctx}/cell/fault/queryDevicePageList.action',
			enbRightUrl:'',
			upsRightUrl:'',
			cpeRightUrl:'',
			gnbRightUrl:'',
			egwRightUrl:'',
            gsmRightUrl:'',
			ruleForm:{
				ruleName:'',
				status:'0',
				ruleType:'1',
				deviceType:'ENB',
				deviceTypeList:['ENB'],
				enbDeviceByDeviceGroup:1,
				upsDeviceByDeviceGroup:1,
				cpeDeviceByDeviceGroup:1,
				gnbDeviceByDeviceGroup:1,
				egwDeviceByDeviceGroup:1,
                gsmDeviceByDeviceGroup:1,

				enbAllAndCustomize:1,
				upsAllAndCustomize:1,
				cpeAllAndCustomize:1,
				gnbAllAndCustomize:1,
				egwAllAndCustomize:1,
                gsmAllAndCustomize:1,

				enbCode:'',
				upsCode:'',
				cpeCode:'',
				gnbCode:'',
				egwCode:'',
                gsmCode:'',

				alarmId:'',
				enbGroupId:'',
				upsGroupId:'',
				cpeGroupId:'',
				gnbGroupId:'',
				egwGroupId:'',
                gsmGroupId:'',
				
				startTime:'',
				endTime:'',
				allAlarm:0,
				timeZone:timeZone,
			},
			rules:{
				ruleName:[
					{validator:validateName,trigger:'blur'}
				],
				deviceTypeList:[
					{type:'array',required:true, message:'<%=rb.getString("SheBeiLeiXingWeiKongTiShi")%>',trigger:'change'}
				],
				alarmId:[
					{validator:validatorAlarmId,trigger:'change'}
				],
			},
			curGroupRow: '',
			upsCurGroupRow:'',
			cpeCurGroupRow:'',
			gnbCurGroupRow:'',
			egwCurGroupRow:'',
            gsmCurGroupRow:'',
			
			alarmTabsShow:true,
			deviceTypeShowENB:true,
			deviceTypeShowUPS:false,
			deviceTypeShowCPE:false,
			deviceTypeShowGNB:false,
			deviceTypeShowEGW:false,
            deviceTypeShowGSM:false,
			
			tableStatus:{
				enbDeviceGroup:{loaded:false,isFirst:0},
				enbDevice:{loaded:false,isFirst:0},
				upsDeviceGroup:{loaded:false,isFirst:0},
				upsDevice:{loaded:false,isFirst:0},
				cpeDeviceGroup:{loaded:false,isFirst:0},
				cpeDevice:{loaded:false,isFirst:0},
				gnbDeviceGroup:{loaded:false,isFirst:0},
				gnbDevice:{loaded:false,isFirst:0},
				egwDeviceGroup:{loaded:false,isFirst:0},
				egwDevice:{loaded:false,isFirst:0},
                gsmDeviceGroup:{loaded:false,isFirst:0},
                gsmDevice:{loaded:false,isFirst:0},
				alarmList:{loaded:false,isFirst:0},
			},
		}
	},
	watch:{
		// enb设备组列表已选中列表监测
		enbDeviceGroupSelection(){
			var data = this.enbDeviceGroupSelection;
			let device_group = ""
			data.map(function(item){
				device_group += item.id+','
			})
			this.ruleForm.enbGroupId = device_group;
		},
		// ups设备组列表已选中列表监测
		upsDeviceGroupSelection(){
			var data = this.upsDeviceGroupSelection;
			let device_group = ""
			data.map(function(item){
				device_group += item.id+','
			})
			this.ruleForm.upsGroupId = device_group;
		},
		// cpe设备组列表已选中列表监测
		cpeDeviceGroupSelection(){
			var data = this.cpeDeviceGroupSelection;
			let device_group = ""
			data.map(function(item){
				device_group += item.id+','
			})
			this.ruleForm.cpeGroupId = device_group;
		},
		// gnb设备组列表已选中列表监测
		gnbDeviceGroupSelection(){
			var data = this.gnbDeviceGroupSelection;
			let device_group = ""
			data.map(function(item){
				device_group += item.id+','
			})
			this.ruleForm.gnbGroupId = device_group;
		},
		// egw设备组列表已选中列表监测
		egwDeviceGroupSelection(){
			var data = this.egwDeviceGroupSelection;
			let device_group = ""
			data.map(function(item){
				device_group += item.id+','
			})
			this.ruleForm.egwGroupId = device_group;
		},
		// gsm设备组列表已选中列表监测
        gsmDeviceGroupSelection(){ 
            var data = this.gsmDeviceGroupSelection;
            let device_group = ""
            data.map(function(item){
                device_group += item.id+','
            })
            this.ruleForm.gsmGroupId = device_group;
		},
		// 基站设备列表已选中列表监测
		enbDeviceSelection(){
			var data = this.$refs.cpairgrid.getData();
			var deviceCode = '';
			data.map(function(item){
				deviceCode += item.small_cell_code + ","
			})
			this.ruleForm.enbCode = deviceCode;
		},
		// UPS设备列表已选中列表监测
		upsDeviceSelection(){ 
			var data = this.$refs.cpairgridUPS.getData();
			var deviceCode = '';
			data.map(function(item){
				deviceCode += item.ups_code + ","
			})
			this.ruleForm.upsCode = deviceCode;
		},
		// CPE设备列表已选中列表监测
		cpeDeviceSelection(){ 
			var data = this.$refs.cpairgridCPE.getData();
			var deviceCode = '';
			data.map(function(item){
				deviceCode += item.cpe_code + ","
			})
			this.ruleForm.cpeCode = deviceCode;
		},
		// GNB设备列表已选中列表监测
		gnbDeviceSelection(){ 
			var data = this.$refs.cpairgridGNB.getData();
			var deviceCode = '';
			data.map(function(item){
				deviceCode += item.small_cell_code + ","
			})
			this.ruleForm.gnbCode = deviceCode;
		},
		// egw设备列表已选中列表监测
		egwDeviceSelection(){ 
			var data = this.$refs.cpairgridEGW.getData();
			var deviceCode = '';
			data.map(function(item){
				deviceCode += item.egw_code + ","
			})
			this.ruleForm.egwCode = deviceCode;
		},
        // gsm设备列表已选中列表监测
        gsmDeviceSelection(){ 
            var data = this.$refs.cpairgridGSM.getData();
            var deviceCode = '';
            data.map(function(item){
                deviceCode += item.small_cell_code + ","
            })
            this.ruleForm.gsmCode = deviceCode;
        },
		"ruleForm.deviceType":function(newValue,oldValue){
			this.alarmLeftUrl = '${ctx}/cell/fault/queryAlarmLevelInfosList.action?deviceType='+newValue+'&timeZone='+timeZone;
			if(this.actionType == 'edit' ){
				this.alarmRightUrl = "${ctx}/cell/fault/queryRuleIdentifierById.action?ruleId="+this.ruleId+"&deviceType="+newValue;
			}
		},
		"ruleForm.deviceTypeList":function(newValue,oldValue){
			this.ruleForm.deviceType = newValue.join(',');
			var codes=['ENB','UPS','CPE','GNB','EGW','GSM'], 
				types={
					ENB:{
						values:[true,'','',1,false],
						keys:['deviceTypeShowENB','enbGroupId','enbCode','enbDeviceByDeviceGroup']
					},
					UPS:{
						values:[true,'','',1,false],
						keys:['deviceTypeShowUPS','upsGroupId','upsCode','upsDeviceByDeviceGroup']
					},
					CPE:{
						values:[true,'','',1,false],
						keys:['deviceTypeShowCPE','cpeGroupId','cpeCode','cpeDeviceByDeviceGroup']
					},
					GNB:{
						values:[true,'','',1,false],
						keys:['deviceTypeShowGNB','gnbGroupId','gnbCode','gnbDeviceByDeviceGroup']
					},
					EGW:{
						values:[true,'','',1,false],
						keys:['deviceTypeShowEGW','egwGroupId','egwCode','egwDeviceByDeviceGroup']
					},
                    GSM:{
                        values:[true,'','',1,false],
                        keys:['deviceTypeShowGSM','gsmGroupId','gsmCode','gsmDeviceByDeviceGroup']
                    },
				};
			var vm = this,
				alarmTypeCodes = [];
			vm.initAlarmTypeArr.map((item)=>{
				alarmTypeCodes.push(item.value);
			})
			codes.map(function(code){

				if(newValue.includes(code) && alarmTypeCodes.includes(code)){
					vm[types[code].keys[0]] = types[code].values[0];
				}else{
					types[code].keys.map(function(key,index){
						vm[types[code].keys[0]] = types[code].values[4];
						if(index>0){
							vm.ruleForm[key] = types[code].values[index];
						}
					})
				}
				
			})

			if(newValue.length !== 0){
				if(newValue.indexOf('ENB') !== -1){
					this.alarmTabsValue = 'ENB';
					this.alarmTabsShow = true;
					return
				}
				if(newValue.indexOf('UPS') !== -1){
					this.alarmTabsValue = 'UPS';
					this.alarmTabsShow = true;
					return
				}
				if(newValue.indexOf('CPE') !== -1){
					this.alarmTabsValue = 'CPE';
					this.alarmTabsShow = true;
					return
				}
				if(newValue.indexOf('GNB') !== -1){
					this.alarmTabsValue = 'GNB';
					this.alarmTabsShow = true;
					return
				}
				if(newValue.indexOf('EGW') !== -1){
					this.alarmTabsValue = 'EGW';
					this.alarmTabsShow = true;
					return
				}
                if(newValue.indexOf('GSM') !== -1){
                    this.alarmTabsValue = 'GSM';
                    this.alarmTabsShow = true;
                    return
                }
				if(newValue.indexOf('OMC') !== -1){
					this.alarmTabsShow = false;
				}
			}else{
				this.alarmTabsShow = false;
			}
		},
		"ruleForm.enbDeviceByDeviceGroup":function(newValue,oldValue){
			if(newValue == 0){
				this.ruleForm.enbGroupId = '';
			}else{
				this.ruleForm.enbCode = '';
			}
		},
		"ruleForm.upsDeviceByDeviceGroup":function(newValue,oldValue){
			if(newValue == 0){
				this.ruleForm.upsGroupId = '';
			}else{
				this.ruleForm.upsCode = '';
			}
		},
		"ruleForm.cpeDeviceByDeviceGroup":function(newValue,oldValue){
			if(newValue == 0){
				this.ruleForm.cpeGroupId = '';
			}else{
				this.ruleForm.cpeCode = '';
			}
		},
		"ruleForm.gnbDeviceByDeviceGroup":function(newValue,oldValue){
			if(newValue == 0){
				this.ruleForm.gnbGroupId = '';
			}else{
				this.ruleForm.gnbCode = '';
			}
		},
		"ruleForm.egwDeviceByDeviceGroup":function(newValue,oldValue){
			if(newValue == 0){
				this.ruleForm.egwGroupId = '';
			}else{
				this.ruleForm.egwCode = '';
			}
		},
        "ruleForm.gsmDeviceByDeviceGroup":function(newValue,oldValue){
            if(newValue == 0){
                this.ruleForm.gsmGroupId = '';
            }else{
                this.ruleForm.gsmCode = '';
            }
        },
	},
	computed: {
		alarmSourceList(){
			var alarmSourceArr = [
				{label:'All',value:''}
			];
			if(this.ruleForm.deviceTypeList.length>0){
				this.ruleForm.deviceTypeList.map((item)=>{
					if(item == 'EGW'){
						alarmSourceArr.push({label:'WCG',value:item})
					}else{
						alarmSourceArr.push({label:item,value:item})
					}
				})
			}
			return alarmSourceArr
		}
	},
	methods:{
		// 表格数据加载成功回调
		tableLoadSuccess(val){
			var vm = this,
				types={
					enbDeviceGroup:{
						all:'enbAllAndCustomize',
						ref:'deviceGroupTable'
					},
					upsDeviceGroup:{
						all:'upsAllAndCustomize',
						ref:'upsDeviceGroupTable'
					},
					cpeDeviceGroup:{
						all:'cpeAllAndCustomize',
						ref:'cpeDeviceGroupTable'
					},
					gnbDeviceGroup:{
						all:'gnbAllAndCustomize',
						ref:'gnbDeviceGroupTable'
					},
					egwDeviceGroup:{
						all:'egwAllAndCustomize',
						ref:'egwDeviceGroupTable'
					},
                    gsmDeviceGroup:{
                        all:'gsmAllAndCustomize',
                        ref:'gsmDeviceGroupTable'
                    },
				};
			for(var key in types){
				var allCus = types[key].all,
					refKey = types[key].ref;
				if(key == val && vm.ruleForm[allCus] == 0){
					var rows = vm.$refs[refKey].getData();
					rows.map(function(row){
						vm.$refs[refKey].toggleRowSelection(row, true);
					})
				}
			};
			if(val == 'alarmList'){
				if(vm.ruleForm.allAlarm == 1){
					vm.$refs.calarmPairgrids.toggleAllSelection();
				}else{
					vm.alarmSelectChange();
				}
			}
			var codes = ['ENB','UPS','CPE','GNB','EGW','GSM'], 
				refs = {
					ENB:{
						by:'enbDeviceByDeviceGroup',
						key:'enbDevice',
						groupKey:'enbDeviceGroup'
					},
					UPS:{
						by:'upsDeviceByDeviceGroup',
						key:'upsDevice',
						groupKey:'upsDeviceGroup'
					},
					CPE:{
						by:'cpeDeviceByDeviceGroup',
						key:'cpeDevice',
						groupKey:'cpeDeviceGroup'
					},
					GNB:{
						by:'gnbDeviceByDeviceGroup',
						key:'gnbDevice',
						groupKey:'gnbDeviceGroup'
					},
					EGW:{
						by:'egwDeviceByDeviceGroup',
						key:'egwDevice',
						groupKey:'egwDeviceGroup'
					},
                    GSM:{
                        by:'gsmDeviceByDeviceGroup',
                        key:'gsmDevice',
                        groupKey:'gsmDeviceGroup'
                    },
				}
			codes.map(function(code){
				var byKey = refs[code].by,
					key = refs[code].key,
					groupKey = refs[code].groupKey;
				
				if(vm.ruleForm.deviceTypeList.includes(code)){
					if(vm.ruleForm[byKey] == 1){
						vm.tableStatus[key].loaded = true;
					}else{
						vm.tableStatus[groupKey].loaded = true;
					}
				}else{
					vm.tableStatus[key].loaded = true;
					vm.tableStatus[groupKey].loaded = true;
				}
			})

			if(vm.actionType == 'add'){
				vm.tableStatus['alarmList'].loaded = true;
			}
			vm.tableStatus[val].loaded = true;

			var isAllReady = true,
				boolList = [],
				numList = [];
			for(var key in vm.tableStatus){
				var item = vm.tableStatus[key];
				boolList.push(item.loaded);
				numList.push(item.isFirst);
			}
			if(boolList.includes(false)){
				isAllReady = false;
				
			}else{
				isAllReady = true;
			}
			if(isAllReady&&(vm.actionType == 'edit' || vm.actionType == 'add')){
				vm.$nextTick(function(){
					initForm(vm.$refs.ruleForm);
				})
			}
			
		},
		/**
		* 设备类型切换
		* @param val{array}  设备类型列表
		*/
		deviceTypeListChange(val){
			var vm = this;
			vm.ruleForm.deviceType = val.join(',');
			if(vm.ruleForm.allAlarm == 0){
				vm.$refs.calarmPairgrid.clear();
			}
		},
		// tab 移除事件
		removeTab(val){
			var vm = this,idx;
			if(vm.actionType == 'view')return
			idx = vm.ruleForm.deviceTypeList.findIndex((item)=>item === val);
			vm.ruleForm.deviceTypeList.splice(idx,1);

			if(vm.alarmQueryForm.alarmSource && vm.alarmQueryForm.alarmSource == val){
				vm.alarmQueryForm.alarmSource = '';
				vm.alarmQueryFormData.alarmSource = '';
			}
			if(vm.alarmQueryFormAll.alarmSource && vm.alarmQueryFormAll.alarmSource == val){
				vm.alarmQueryFormAll.alarmSource = '';
				vm.alarmQueryFormDataAll.alarmSource = '';
			}
			if(vm.ruleForm.deviceTypeList.length !== 0 ){
				if(vm.ruleForm.deviceTypeList.indexOf('ENB') !== -1){
					vm.alarmTabsValue = 'ENB';
					vm.alarmTabsShow = true;
					return
				}
				if(vm.ruleForm.deviceTypeList.indexOf('UPS') !== -1){
					vm.alarmTabsValue = 'UPS';
					vm.alarmTabsShow = true;
					return
				}
				if(vm.ruleForm.deviceTypeList.indexOf('CPE') !== -1){
					vm.alarmTabsValue = 'CPE';
					vm.alarmTabsShow = true;
					return
				}
				if(vm.ruleForm.deviceTypeList.indexOf('GNB') !== -1){
					vm.alarmTabsValue = 'GNB';
					vm.alarmTabsShow = true;
					return
				}
				if(vm.ruleForm.deviceTypeList.indexOf('EGW') !== -1){
					vm.alarmTabsValue = 'EGW';
					vm.alarmTabsShow = true;
					return
				}
                if(vm.ruleForm.deviceTypeList.indexOf('GSM') !== -1){
                    vm.alarmTabsValue = 'GSM';
                    vm.alarmTabsShow = true;
                    return
                }
				if(vm.ruleForm.deviceTypeList.indexOf('OMC') !== -1){
					vm.alarmTabsShow = false;
				}
			}else{
				vm.alarmTabsShow = false;
			}
		},
		queryUPS(){
			this.$refs.cpairgridUPS.reload();
		},
		// 基站列表查询
		query(){ 
			this.$refs.cpairgrid.reload();
		},
		queryCPE(){
			this.$refs.cpairgridCPE.reload();
		},
		queryGNB(){
			this.$refs.cpairgridGNB.reload();
		},
		queryEGW(){
			this.$refs.cpairgridEGW.reload();
		},
        queryGSM(){
            this.$refs.cpairgridGSM.reload();
        },
		// 告警模糊查询
		queryAlarm(val){
			var vm =  this;
			vm.alarmReset();
			vm.alarmQueryForm.search_text = val;
		},
		// 告警列表 高级查询
		alarmQueryAdvance(){
			var vm =  this;
			Object.assign(vm.alarmQueryForm, vm.alarmQueryFormData);
		},
		//ALL 全选列表 模糊查询 
		queryAlarmAll(val){
			var vm =  this;
			vm.alarmQueryFormAll.search_text = val;
		},
		//ALL 全选列表  高级查询
		alarmQueryAdvanceAll(){
			var vm =  this;
			Object.assign(vm.alarmQueryFormAll, vm.alarmQueryFormDataAll);
		},
		// 告警列表 查询重置
		alarmReset(){
			var vm = this,
				params={
					alarmIdentifier:'',
					alarmType:'',
					alarmServerity:'',
					alarmSource:'',
				};
			Object.assign(vm.alarmQueryFormData, params);
			Object.assign(vm.alarmQueryForm, params);
		},
		// ALL 全选列表 查询重置
		alarmResetAll(){
			var vm = this,
				params={
					alarmIdentifier:'',
					alarmType:'',
					alarmServerity:'',
					alarmSource:'',
				};
			Object.assign(vm.alarmQueryFormDataAll, params);
			Object.assign(vm.alarmQueryFormAll, params);
		},
		// 基站 设备组，设备切换事件
		enbDeviceByDeviceGroupChange(val){
			var vm = this;
			
			if(val == 0){
				vm.$refs.deviceGroupTable.clearSelection();
			}else{
				vm.$refs.cpairgrid.clear();
			}
		},
		// ups 设备组，设备切换事件
		upsDeviceByDeviceGroupChange(val){
			var vm = this;

			if(val == 0){
				vm.$refs.upsDeviceGroupTable.clearSelection();
			}else{
				vm.$refs.cpairgridUPS.clear();
			}
		},
		// cpe 设备组，设备切换事件
		cpeDeviceByDeviceGroupChange(val){
			var vm = this;
			
			if(val == 0){
				vm.$refs.cpeDeviceGroupTable.clearSelection();
			}else{
				vm.$refs.cpairgridCPE.clear();
			}
		},
		// gnb 设备组，设备切换事件
		gnbDeviceByDeviceGroupChange(val){
			var vm = this;
			
			if(val == 0){
				vm.$refs.gnbDeviceGroupTable.clearSelection();
			}else{
				vm.$refs.cpairgridGNB.clear();
			}
		},
		// EGW 设备组，设备切换事件
		egwDeviceByDeviceGroupChange(val){
			var vm = this;
			
			if(val == 0){
				vm.$refs.egwDeviceGroupTable.clearSelection();
			}else{
				vm.$refs.cpairgridEGW.clear();
			}
		},
        // GSM 设备组，设备切换事件
        gsmDeviceByDeviceGroupChange(val){ 
            var vm = this;
            
            if(val == 0){
                vm.$refs.gsmDeviceGroupTable.clearSelection();
            }else{
                vm.$refs.cpairgridGSM.clear();
            }
        },
		/**
		* 右上角关闭按钮
		*/
		closeAddViewTemple(flag){ 
			//发送数据
			var vm = this;
			eventBus.$emit("close-filter-add");
			eventBus.$emit("init-ruleTable");
		},
		/**
		* enb设备组单行点击事件
		* @param row{object}  行数据
		* @param col{object}  列数据
		* @param evt{object}  event数据
		*/
		groupRowClick(row,evt,col) {  // 设备组单行点击事件
			var vm = this;
			if(vm.curGroupRow && vm.curGroupRow.id == row.id) {
				vm.$refs.groupFiltor.setCurrentRow();
				vm.queryForm.groupId = '';
				vm.curGroupRow = '';
			}else {
				vm.queryForm.groupId = row.id;
				vm.curGroupRow = row;
			}
		},
		/**
		* ups设备组单行点击事件
		* @param row{object}  行数据
		* @param col{object}  列数据
		* @param evt{object}  event数据
		*/
		upsGroupRowClick(row,evt,col) {  // 设备组单行点击事件
			var vm = this;
			if(vm.upsCurGroupRow && vm.upsCurGroupRow.id == row.id) {
				vm.$refs.upsGroupFiltor.setCurrentRow();
				vm.queryFormUPS.groupId = '';
				vm.upsCurGroupRow = '';
			}else {
				vm.queryFormUPS.groupId = row.id;
				vm.upsCurGroupRow = row;
			}
		},
		/**
		* cpe设备组单行点击事件
		* @param row{object}  行数据
		* @param col{object}  列数据
		* @param evt{object}  event数据
		*/
		cpeGroupRowClick(row,evt,col) {  // 设备组单行点击事件
			var vm = this;
			if(vm.cpeCurGroupRow && vm.cpeCurGroupRow.id == row.id) {
				vm.$refs.cpeGroupFiltor.setCurrentRow();
				vm.queryFormCPE.groupId = '';
				vm.cpeCurGroupRow = '';
			}else {
				vm.queryFormCPE.groupId = row.id;
				vm.cpeCurGroupRow = row;
			}
		},
		/**
		* gnb设备组单行点击事件
		* @param row{object}  行数据
		* @param col{object}  列数据
		* @param evt{object}  event数据
		*/
		gnbGroupRowClick(row,evt,col) {  // 设备组单行点击事件
			var vm = this;
			if(vm.gnbCurGroupRow && vm.gnbCurGroupRow.id == row.id) {
				vm.$refs.gnbGroupFiltor.setCurrentRow();
				vm.queryFormGNB.groupId = '';
				vm.gnbCurGroupRow = '';
			}else {
				vm.queryFormGNB.groupId = row.id;
				vm.gnbCurGroupRow = row;
			}
		},
		/**
		* egw设备组单行点击事件
		* @param row{object}  行数据
		* @param col{object}  列数据
		* @param evt{object}  event数据
		*/
		egwGroupRowClick(row,evt,col) {  // 设备组单行点击事件
			var vm = this;
			if(vm.egwCurGroupRow && vm.egwCurGroupRow.id == row.id) {
				vm.$refs.egwGroupFiltor.setCurrentRow();
				vm.queryFormEGW.groupId = '';
				vm.egwCurGroupRow = '';
			}else {
				vm.queryFormEGW.groupId = row.id;
				vm.egwCurGroupRow = row;
			}
		},
        /**
		* GSM设备组单行点击事件
		* @param row{object}  行数据
		* @param col{object}  列数据
		* @param evt{object}  event数据
		*/
        gsmGroupRowClick(row,evt,col) {  // 设备组单行点击事件
            var vm = this;
            if(vm.gsmCurGroupRow && vm.gsmCurGroupRow.id == row.id) {
                vm.$refs.gsmGroupFiltor.setCurrentRow();
                vm.queryFormGSM.groupId = '';
                vm.gsmCurGroupRow = '';
            }else {
                vm.queryFormGSM.groupId = row.id;
                vm.gsmCurGroupRow = row;
            }
        },
		/**
		* 基站设备列表选中数据
		* @param selection{Array}  
		*/
		enbSelectChange(selection){
			this.enbDeviceSelection = selection 
		},
		/**
		* ups设备列表选中数据
		* @param selection{Array}  
		*/
		upsSelectChange(selection){
			this.upsDeviceSelection = selection 
		},
		/**
		* cpe设备列表选中数据
		* @param selection{Array}  
		*/
		cpeSelectChange(selection){
			this.cpeDeviceSelection = selection 
		},
		/**
		* gnb设备列表选中数据
		* @param selection{Array}  
		*/
		gnbSelectChange(selection){
			this.gnbDeviceSelection = selection 
		},
		/**
		* egw 设备列表选中数据
		* @param selection{Array}  
		*/
		egwSelectChange(selection){
			this.egwDeviceSelection = selection 
		},
        /**
		* gsm 设备列表选中数据
		* @param selection{Array}  
		*/
        gsmSelectChange(selection){
			this.gsmDeviceSelection = selection 
		},
		/**
		* enb设备组列表选中数据
		* @param currentRow{Array}  
		*/
		enbDeviceGroupSelect(currentRow){
			this.enbDeviceGroupSelection = currentRow;
		},
		/**
		* ups设备组列表选中数据
		* @param currentRow{Array}  
		*/
		upsDeviceGroupSelect(currentRow){ 
			this.upsDeviceGroupSelection = currentRow ;
		},
		/**
		* cpe设备组列表选中数据
		* @param currentRow{Array}  
		*/
		cpeDeviceGroupSelect(currentRow){
			this.cpeDeviceGroupSelection = currentRow ;
		},
		/**
		* gnb设备组列表选中数据
		* @param currentRow{Array}  
		*/
		gnbDeviceGroupSelect(currentRow){
			this.gnbDeviceGroupSelection = currentRow ;
		},
		/**
		* egw 设备组列表选中数据
		* @param currentRow{Array}  
		*/
		egwDeviceGroupSelect(currentRow){
			this.egwDeviceGroupSelection = currentRow ;
		},
        /**
		* gsm 设备组列表选中数据
		* @param currentRow{Array}  
		*/
        gsmDeviceGroupSelect(currentRow){
			this.gsmDeviceGroupSelection = currentRow ;
		},
		/**
		* 告警列表选中数据
		* @param selection{Array}  
		*/
		alarmSelectChange(){
			var vm = this;
			var alarmIds = '';
			var data = vm.$refs.calarmPairgrid.getData();
			if(data.length != 0){
				data.map(function(item){
					alarmIds += item.ALARM_IDENTIFIER + ","
				})
			}
			vm.ruleForm.alarmId = alarmIds;
		},
		// 新增提交
		submit(){  
			var vm = this,urls;
            // 防止多次点击提交
            if(alarmViewVue.alarmFilterAddSubmitLoading)return

			vm.$refs["ruleForm"].validate( valid => {
				if(valid){
					let params = {};
					params = Object.assign({},vm.ruleForm);
					if(vm.defaultVal !== '1'){
						var deviceTypeByDeviceGroup = [],deviceTypeByDevice = [],allAndCustomize = [];
						vm.ruleForm.deviceTypeList.map((item)=>{
							if(item == 'ENB'){
								if(vm.ruleForm.enbAllAndCustomize == 0){
									allAndCustomize.push('ENB');
									params.enbGroupId = '';
								}else{
									if(vm.ruleForm.enbDeviceByDeviceGroup == 1){
										deviceTypeByDeviceGroup.push('ENB');
									}else{
										deviceTypeByDevice.push('ENB')
										var data = this.$refs.cpairgrid.getData();
										var deviceCodeList = [];
										data.map(function(item){
											deviceCodeList.push(item.small_cell_code)
										})
										params.enbCode = deviceCodeList.join(',');
									}
								}
								
							}
							if(item == 'UPS'){
								if(vm.ruleForm.upsAllAndCustomize == 0){
									allAndCustomize.push('UPS');
									params.upsGroupId = '';
								}else{
									if(vm.ruleForm.upsDeviceByDeviceGroup == 1){
										deviceTypeByDeviceGroup.push('UPS');
									}else{
										deviceTypeByDevice.push('UPS')
										var data = this.$refs.cpairgridUPS.getData();
										var deviceCodeList = [];
										data.map(function(item){
											deviceCodeList.push(item.ups_code)
										})
										params.upsCode = deviceCodeList.join(',');
									}
								}
							}
							if(item == 'CPE'){
								if(vm.ruleForm.cpeAllAndCustomize == 0){
									allAndCustomize.push('CPE');
									params.cpeGroupId = '';
								}else{
									if(vm.ruleForm.cpeDeviceByDeviceGroup == 1){
										deviceTypeByDeviceGroup.push('CPE');
									}else{
										deviceTypeByDevice.push('CPE')
										var data = this.$refs.cpairgridCPE.getData();
										var deviceCodeList = [];
										data.map(function(item){
											deviceCodeList.push(item.cpe_code)
										})
										params.cpeCode = deviceCodeList.join(',');
									}
								}
							}
							if(item == 'GNB'){
								if(vm.ruleForm.gnbAllAndCustomize == 0){
									allAndCustomize.push('GNB');
									params.gnbGroupId = '';
								}else{
									if(vm.ruleForm.gnbDeviceByDeviceGroup == 1){
										deviceTypeByDeviceGroup.push('GNB');
									}else{
										deviceTypeByDevice.push('GNB')
										var data = this.$refs.cpairgridGNB.getData();
										var deviceCodeList = [];
										data.map(function(item){
											deviceCodeList.push(item.small_cell_code)
										})
										params.gnbCode = deviceCodeList.join(',');
									}
								}
							}
							if(item == 'EGW'){
								if(vm.ruleForm.egwAllAndCustomize == 0){
									allAndCustomize.push('EGW');
									params.egwGroupId = '';
								}else{
									if(vm.ruleForm.egwDeviceByDeviceGroup == 1){
										deviceTypeByDeviceGroup.push('EGW');
									}else{
										deviceTypeByDevice.push('EGW')
										var data = this.$refs.cpairgridEGW.getData();
										var deviceCodeList = [];
										data.map(function(item){
											deviceCodeList.push(item.egw_code)
										})
										params.egwCode = deviceCodeList.join(',');
									}
								}
							}
                            if(item == 'GSM'){
                                if(vm.ruleForm.gsmAllAndCustomize == 0){
                                    allAndCustomize.push('GSM');
                                    params.gsmGroupId = '';
                                }else{
                                    if(vm.ruleForm.gsmDeviceByDeviceGroup == 1){
                                        deviceTypeByDeviceGroup.push('GSM');
                                    }else{
                                        deviceTypeByDevice.push('GSM')
                                        var data = this.$refs.cpairgridGSM.getData();
                                        var deviceCodeList = [];
                                        data.map(function(item){
                                            deviceCodeList.push(item.small_cell_code)
                                        })
                                        params.gsmCode = deviceCodeList.join(',');
                                    }
                                }
                            }
						})
						params.deviceTypeByDeviceGroup = deviceTypeByDeviceGroup.join(',');
						params.deviceTypeWithAllDevice = allAndCustomize.join(',');
						params.deviceTypeByDevice = deviceTypeByDevice.join(',');
					}
						
					if(vm.actionType == 'add'){
						urls = "${ctx}/cell/fault/saveRule.action";
					}else if(vm.actionType == 'edit'){
						urls = "${ctx}/cell/fault/modifyRule.action";
						params.ruleId = vm.ruleId;
					}
					delete params.deviceTypeList
					delete params.enbDeviceByDeviceGroup
					delete params.upsDeviceByDeviceGroup
					delete params.cpeDeviceByDeviceGroup
					delete params.gnbDeviceByDeviceGroup
					delete params.egwDeviceByDeviceGroup
                    delete params.gsmDeviceByDeviceGroup
					delete params.enbAllAndCustomize
					delete params.upsAllAndCustomize
					delete params.cpeAllAndCustomize
					delete params.gnbAllAndCustomize
					delete params.egwAllAndCustomize
                    delete params.gsmAllAndCustomize
					if(vm.defaultVal !== '1'){
						if(params.startTime != null && params.endTime != null){
							let startTimeDate = new Date(params.startTime);
							let endTimeDate = new Date(params.endTime);
							let time1 = startTimeDate.getTime();
							let time2 = endTimeDate.getTime();
							if(time2-time1<0){//结束时间比开始时间早
								this.$message({
									message:'<%=rb.getString("JieShuShiJianBuNengXiaoYuKaiShiShiJian")%>',
									type:'warning'
								})
								return false;
							}
						} 
					}
                    alarmViewVue.alarmFilterAddSubmitLoading = true;
					axios.post(urls,stringify(params)).then(function(response){
						if(response.data.success){
							vm.$message({
								message:'<%=rb.getString("ChengGong")%>',
								type:'success',
							})
                            vm.closeAddViewTemple(false);
						}else{
							vm.$message({
								message: response.data.message,
								type:'error',
							});
                            alarmViewVue.alarmFilterAddSubmitLoading = false;
						}
		 			})
				}
			}) 
			
		},
		
		/**
		*  查看 / 修改
		* @param curTempId{number}   模板id
		* @param timeZone{number}    时区
		* @param type{number}   判断查看，修改状态
		*/
		initTask(curTempId,timeZone,type){  
			var vm = this;
			vm.showFlag = false;
			vm.actionType = type;
			vm.ruleId = curTempId;
			if(type == "view"){
				vm.viewFlag = true;
			}else if(type == "edit"){
				vm.viewFlag = false;
			}else if(type == "add"){
				vm.deviceGroupUrl='${ctx}/cell/fault/queryDeviceGroupByOperPageList.action?deviceType=ENB';
				vm.upsDeviceGroupUrl='${ctx}/cell/fault/queryDeviceGroupByOperPageList.action?deviceType=UPS';
				vm.cpeDeviceGroupUrl='${ctx}/cell/fault/queryDeviceGroupByOperPageList.action?deviceType=CPE';
				vm.gnbDeviceGroupUrl='${ctx}/cell/fault/queryDeviceGroupByOperPageList.action?deviceType=GNB';
				vm.egwDeviceGroupUrl='${ctx}/cell/fault/queryDeviceGroupByOperPageList.action?deviceType=EGW';
                vm.gsmDeviceGroupUrl='${ctx}/cell/fault/queryDeviceGroupByOperPageList.action?deviceType=GSM';
				return
			}
			axios.post("${ctx}/cell/fault/queryRuleInfoById.action",stringify({ruleId:curTempId,timeZone:timeZone})).then(function(response){
				var data = response.data
				vm.oldTemplateName = data.rule_name;
				vm.defaultVal = data.default;
				vm.ruleForm.ruleName = data.rule_name;
				vm.ruleForm.status = data.status;
				vm.ruleForm.ruleType = data.rule_type;
				vm.ruleForm.startTime=data.start_time;
				vm.ruleForm.endTime=data.end_time;
				vm.ruleForm.deviceType = data.device_type;
				vm.ruleForm.allAlarm = data.allAlarm;
				
				if(data.enbGroupId){
					vm.enbDeviceChecked = data.enbGroupId;
					vm.ruleForm.enbGroupId = data.enbGroupId.join(",");
				}
				if(data.upsGroupId){
					vm.upsDeviceChecked = data.upsGroupId;
					vm.ruleForm.upsGroupId = data.upsGroupId.join(",");
				}
				if(data.cpeGroupId){
					vm.cpeDeviceChecked = data.cpeGroupId;
					vm.ruleForm.cpeGroupId = data.cpeGroupId.join(",");
				}
				if(data.gnbGroupId){
					vm.gnbDeviceChecked = data.gnbGroupId;
					vm.ruleForm.gnbGroupId = data.gnbGroupId.join(",");
				}
				if(data.egwGroupId){
					vm.egwDeviceChecked = data.egwGroupId;
					vm.ruleForm.egwGroupId = data.egwGroupId.join(",");
				}
                if(data.gsmGroupId){
                    vm.gsmDeviceChecked = data.gsmGroupId;
                    vm.ruleForm.gsmGroupId = data.gsmGroupId.join(",");
                }

				vm.ruleForm.deviceTypeList = data.device_type.split(',');
				if(data.deviceTypeByDeviceGroup.indexOf('ENB') == -1){
					vm.ruleForm.enbDeviceByDeviceGroup = 0
					vm.enbRightUrl="${ctx}/cell/fault/queryRuleDeviceById.action?ruleId="+curTempId+"&deviceType=ENB"
				}else{
					vm.ruleForm.enbDeviceByDeviceGroup = 1
				}
				if(data.deviceTypeByDeviceGroup.indexOf('UPS') == -1){
					vm.ruleForm.upsDeviceByDeviceGroup = 0
					vm.upsRightUrl="${ctx}/cell/fault/queryRuleDeviceById.action?ruleId="+curTempId+"&deviceType=UPS"
				}else{
					vm.ruleForm.upsDeviceByDeviceGroup = 1
				}
				if(data.deviceTypeByDeviceGroup.indexOf('CPE') == -1){
					vm.ruleForm.cpeDeviceByDeviceGroup = 0
					vm.cpeRightUrl="${ctx}/cell/fault/queryRuleDeviceById.action?ruleId="+curTempId+"&deviceType=CPE"
				}else{
					vm.ruleForm.cpeDeviceByDeviceGroup = 1
				}
				if(data.deviceTypeByDeviceGroup.indexOf('GNB') == -1){
					vm.ruleForm.gnbDeviceByDeviceGroup = 0
					vm.gnbRightUrl="${ctx}/cell/fault/queryRuleDeviceById.action?ruleId="+curTempId+"&deviceType=GNB"
				}else{
					vm.ruleForm.gnbDeviceByDeviceGroup = 1
				}
				if(data.deviceTypeByDeviceGroup.indexOf('EGW') == -1){
					vm.ruleForm.egwDeviceByDeviceGroup = 0
					vm.egwRightUrl="${ctx}/cell/fault/queryRuleDeviceById.action?ruleId="+curTempId+"&deviceType=EGW"
				}else{
					vm.ruleForm.egwDeviceByDeviceGroup = 1
				}
                if(data.deviceTypeByDeviceGroup.indexOf('GSM') == -1){
                    vm.ruleForm.gsmDeviceByDeviceGroup = 0
                    vm.gsmRightUrl="${ctx}/cell/fault/queryRuleDeviceById.action?ruleId="+curTempId+"&deviceType=GSM"
                }else{
                    vm.ruleForm.gsmDeviceByDeviceGroup = 1
                }

				if(data.deviceTypeWithAllDevice.indexOf('ENB') == -1){
					vm.ruleForm.enbAllAndCustomize = 1;
					setTimeout(function(){
						vm.deviceGroupUrl='${ctx}/cell/fault/queryDeviceGroupByOperPageList.action?deviceType=ENB';
					},200);
				}else{
					vm.ruleForm.enbAllAndCustomize = 0;
					vm.ruleForm.enbDeviceByDeviceGroup = 1;
					setTimeout(function(){
						vm.deviceGroupUrl='${ctx}/cell/fault/queryDeviceGroupByOperPageList.action?deviceType=ENB';
					},200);
				}
				if(data.deviceTypeWithAllDevice.indexOf('UPS') == -1){
					vm.ruleForm.upsAllAndCustomize = 1;
					setTimeout(function(){
						vm.upsDeviceGroupUrl='${ctx}/cell/fault/queryDeviceGroupByOperPageList.action?deviceType=UPS';
					},200);
				}else{
					vm.ruleForm.upsAllAndCustomize = 0;
					vm.ruleForm.upsDeviceByDeviceGroup = 1;
					setTimeout(function(){
						vm.upsDeviceGroupUrl='${ctx}/cell/fault/queryDeviceGroupByOperPageList.action?deviceType=UPS';
					},200);
				}
				if(data.deviceTypeWithAllDevice.indexOf('CPE') == -1){
					vm.ruleForm.cpeAllAndCustomize = 1;
					setTimeout(function(){
						vm.cpeDeviceGroupUrl='${ctx}/cell/fault/queryDeviceGroupByOperPageList.action?deviceType=CPE';
					},200);
				}else{
					vm.ruleForm.cpeAllAndCustomize = 0;
					vm.ruleForm.cpeDeviceByDeviceGroup = 1;
					setTimeout(function(){
						vm.cpeDeviceGroupUrl='${ctx}/cell/fault/queryDeviceGroupByOperPageList.action?deviceType=CPE';
					},200);
				}
				if(data.deviceTypeWithAllDevice.indexOf('GNB') == -1){
					vm.ruleForm.gnbAllAndCustomize = 1;
					setTimeout(function(){
						vm.gnbDeviceGroupUrl='${ctx}/cell/fault/queryDeviceGroupByOperPageList.action?deviceType=GNB';
					},200);
				}else{
					vm.ruleForm.gnbAllAndCustomize = 0;
					vm.ruleForm.gnbDeviceByDeviceGroup = 1;
					setTimeout(function(){
						vm.gnbDeviceGroupUrl='${ctx}/cell/fault/queryDeviceGroupByOperPageList.action?deviceType=GNB';
					},200);
				}
				if(data.deviceTypeWithAllDevice.indexOf('EGW') == -1){
					vm.ruleForm.egwAllAndCustomize = 1;
					setTimeout(function(){
						vm.egwDeviceGroupUrl='${ctx}/cell/fault/queryDeviceGroupByOperPageList.action?deviceType=EGW';
					},200);
				}else{
					vm.ruleForm.egwAllAndCustomize = 0;
					vm.ruleForm.egwDeviceByDeviceGroup = 1;
					setTimeout(function(){
						vm.egwDeviceGroupUrl='${ctx}/cell/fault/queryDeviceGroupByOperPageList.action?deviceType=EGW';
					},200);
				}
                if(data.deviceTypeWithAllDevice.indexOf('GSM') == -1){
                    vm.ruleForm.gsmAllAndCustomize = 1;
                    setTimeout(function(){
                        vm.gsmDeviceGroupUrl='${ctx}/cell/fault/queryDeviceGroupByOperPageList.action?deviceType=GSM';
                    },200);
                }else{
                    vm.ruleForm.gsmAllAndCustomize = 0;
                    vm.ruleForm.gsmDeviceByDeviceGroup = 1;
                    setTimeout(function(){
                        vm.gsmDeviceGroupUrl='${ctx}/cell/fault/queryDeviceGroupByOperPageList.action?deviceType=GSM';
                    },200);
                }

				vm.alarmRightUrl = "${ctx}/cell/fault/queryRuleIdentifierById.action?ruleId="+curTempId+"&deviceType="+data.device_type
				if( type == "edit"){
					initForm(vm.$refs.ruleForm);
				}
			})
		},
		//取消
		cancel(){
			var vm = this;
			var confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>';
			if(vm.actionType == 'edit' || vm.actionType == 'add'){
				if(isFormChanged(this.$refs.ruleForm)){
					vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
						customClass:'warningConfirm',
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(() => {
						vm.closeAddViewTemple(false);
					}).catch(() => {
						
					})
				}else{
					vm.closeAddViewTemple(false);
				}
			}else{
				vm.closeAddViewTemple(false);
			}
			
		},
		//初始化 告警类型值
		initAlarmType(curTempId,timeZone,type){
			var vm = this;
			axios.post("${ctx}/cell/fault/queryAlarmType.action").then((res) => {
				var alarmTypeArr = [],
					data = res.data.AlarmType ? res.data.AlarmType : [];

				data.map((item)=>{
					if(item.toUpperCase() == 'EGW'){
						alarmTypeArr.push({label:'WCG',value:item.toUpperCase()})
					}else{
						alarmTypeArr.push({label:item.toUpperCase(),value:item.toUpperCase()})
					}
				})
				vm.initAlarmTypeArr = alarmTypeArr;
				vm.initTask(curTempId,timeZone,type);
			})
		},
		// 告警列表全选事件
		allAlarmChange(val){
			var vm = this;
			if(val == 1){
				vm.$nextTick(()=>{
					vm.$refs.calarmPairgrids.toggleAllSelection();
				})
			}else{
				vm.$refs.calarmPairgrids.clearSelection();
			}
		},
		// enb设备 设备组全选事件
		enbAllAndCustomizeChange(val){
			var vm = this;
			if(val == 0){
				vm.ruleForm.enbDeviceByDeviceGroup = 1;
				var rows = vm.$refs.deviceGroupTable.getData();
				rows.map(function(row){
					vm.$refs.deviceGroupTable.toggleRowSelection(row, true);
				})
			}else{
				vm.$refs.deviceGroupTable.clearSelection();
			}
			vm.ruleForm.enbGroupId = '';
		},
		// ups设备 设备组全选事件
		upsAllAndCustomizeChange(val){
			var vm = this;
			if(val == 0){
				vm.ruleForm.upsDeviceByDeviceGroup = 1;
				var rows = vm.$refs.upsDeviceGroupTable.getData();
				rows.map(function(row){
					vm.$refs.upsDeviceGroupTable.toggleRowSelection(row, true);
				})
			}else{
				vm.$refs.upsDeviceGroupTable.clearSelection();
			}
			vm.ruleForm.upsGroupId = '';
		},
		// cpe设备 设备组全选事件
		cpeAllAndCustomizeChange(val){
			var vm = this;
			if(val == 0){
				vm.ruleForm.cpeDeviceByDeviceGroup = 1;
				var rows = vm.$refs.cpeDeviceGroupTable.getData();
				rows.map(function(row){
					vm.$refs.cpeDeviceGroupTable.toggleRowSelection(row, true);
				})
			}else{
				vm.$refs.cpeDeviceGroupTable.clearSelection();
			}
			vm.ruleForm.cpeGroupId = '';
		},
		// gnb设备 设备组全选事件
		gnbAllAndCustomizeChange(val){
			var vm = this;
			if(val == 0){
				vm.ruleForm.gnbDeviceByDeviceGroup = 1;
				var rows = vm.$refs.gnbDeviceGroupTable.getData();
				rows.map(function(row){
					vm.$refs.gnbDeviceGroupTable.toggleRowSelection(row, true);
				})
			}else{
				vm.$refs.gnbDeviceGroupTable.clearSelection();
			}
			vm.ruleForm.gnbGroupId = '';
		},
		// egw 设备 设备组全选事件
		egwAllAndCustomizeChange(val){
			var vm = this;
			if(val == 0){
				vm.ruleForm.egwDeviceByDeviceGroup = 1;
				var rows = vm.$refs.egwDeviceGroupTable.getData();
				rows.map(function(row){
					vm.$refs.egwDeviceGroupTable.toggleRowSelection(row, true);
				})
			}else{
				vm.$refs.egwDeviceGroupTable.clearSelection();
			}
			vm.ruleForm.egwGroupId = '';
		},
         // gsm设备 设备组全选事件
        gsmAllAndCustomizeChange(val){ 
            var vm = this;
            if(val == 0){
                vm.ruleForm.gsmDeviceByDeviceGroup = 1;
                var rows = vm.$refs.gsmDeviceGroupTable.getData();
                rows.map(function(row){
                    vm.$refs.gsmDeviceGroupTable.toggleRowSelection(row, true);
                })
            }else{
                vm.$refs.gsmDeviceGroupTable.clearSelection();
            }
            vm.ruleForm.gsmGroupId = '';
        },
	},
	components:{},
	mounted(){
		eventBus.$off('filter-ok-new').$on('filter-ok-new',this.submit);
		eventBus.$off('filter-add-cancel').$on('filter-add-cancel',this.cancel);
		eventBus.$off('rule-info-task').$on('rule-info-task',this.initAlarmType);
	},
})

</script>