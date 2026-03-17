<%@ page import="java.util.Map" %>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%
	HttpSession s = request.getSession();
	String languageCode = (String)s.getAttribute("language_code");
%>
<style type="text/css">
#operatorPage .pageContainer{
	display:flex;
	height:100%;
}
#operatorPage .operatorCon{
	flex:1 1 37%;
	background:#FFF;
}
#operatorPage .operatTableList{
	flex: 1 1 63%;
	background:#ffffff;
	border-left:2px solid #e9e9e9;
	box-sizing:border-box;
}
#operatorPage .tableHeader{
	display:flex;
	box-sizing:border-box;
	position:relative;
	padding:0px 20px;
	align-items:center;
}
#operatorPage .el-slide{
	right:0px !important;
}
#operatorPage .searchCon{
	width:400px;
	margin-left:27px;
}
#operatorPage .searchCon .el-input{
	width:100%;
}
#operatorPage .searchCon .el-input__inner{
	border-radius:4px;
	background:#FFFFFF;
	border: 1px solid #E9E9E9;
	height:30px;
	line-height:30px;
}
#operatorPage .buttonGroup{
	position:absolute;
	right:20px;
	top:0px; 
	z-index:10;
}
#operatorPage .buttonGroup div{
	width:36px;
	float:left;
}
#operatorPage .buttonGroup p{
	text-align:center;
	font-size:16px;
	color:#1DA3FC;
	margin-top:5px;
}
#operatorPage .buttonGroup .el-button:focus,#operatorPage .buttonGroup .el-button:hover{
	background-color:#1DA3FC;
}
#operatorPage .headerTitle{
	font-weight:bold;
	color:#363B4E;
	font-size:16px;
}
#operatorPage .operatorCon .el-dialog{
	width:22%;
}
#operatorPage .el-tabs{
	height:100%;
}
#operatorPage  .el-tabs .el-tabs__item{
	font-size:16px;
	margin: 5px 0px;
}
#operatorPage .operatTableList .el-tabs .el-tabs__item{
	font-size:16px;
	margin: 3px 0px;
}
#operatorPage .select-device-container{
	position:absolute;
	height:60px;
	background:rgba(255,255,255,1);
	box-shadow:0px -5px 10px rgba(0,0,0,0.1);
	bottom:0px;
	left:0px;
	right:0px;
	box-sizing:border-box;
	padding:0px 20px 0px 20px;
	display:flex;
	align-items:center;
	justify-content:space-between;
	z-index:109;
}
#operatorPage .titleSpan{
	font-weight:bold;
	color:#333333;
	font-size:14px;
	align-items:center;
}
#operatorPage .selfbutton{
	height:24px;
	line-height:24px;
	padding:0px 20px;
	margin-right:15px;
}
#operatorPage .el-tabs__nav{
	margin-left:18px;
}
#operatorPage .operatorCon .el-message-box__content,#operatorPage .operatorCon .el-dialog__body{
	padding:30px;
}
#operatorPage .el-dialog__footer{
	padding:0 30px 10px;
	display:flex;
}
.deldialog .el-dialog__footer{
	display:block;
	margin-right:20px;
}
.addOperDia .el-input{
	width:300px;
}
#operatorPage .el-tabs__nav-wrap{
	border-bottom:1px solid #E9E9E9;
}
.deldialog .el-form-item{
	display:flex;
	align-items:center;
	margin-top:15px;
	margin-bottom:0px;
}
#operatorPage .has-select-container{
	position:absolute;
	width:380px;
	height:400px;
	box-sizing:border-box;
	display:flex;
	flex-direction:column;
	bottom:60px;
	left:0px;
	background:#FFFFFF;
	box-shadow:5px -5px 15px rgba(0,0,0,0.1)
}
#operatorPage .select-container-header{
	height:40px;
	border-bottom:1px solid #E0E4ED;
	padding:0px 20px;
	box-sizing:border-box;
	display:flex;
	line-height:40px;
	justify-content: space-between;
}
#operatorPage .border-header-tltle{
	font-weight:bold;
	color: #363B4E;
	font-size: 16px;
}
#operatorPage .select-container-body{
	flex: 1;
	overflow:auto;
	display: flex;
	flex-direction: column;
	box-sizing: border-box;
	padding: 0px 10px;
}
#operatorPage .select-container-body div{
	padding: 0px 10px;
	box-sizing: border-box;
}
#operatorPage .select-body-header{
	height: 46px;
	border-bottom:1px solid #F4F4F4;
	display: flex;
	justify-content: space-between;
	line-height: 46px;

}
#operatorPage .body-header-title{
	font-weight: bold;
	color: #333333;
	font-size: 14px;
}
#operatorPage .body-list{
	flex: 1;
	overflow: auto;
}
#operatorPage .select-body-list{
	height: 36px;
	line-height:36px;
	display: flex;
	justify-content: space-between;
	border-bottom: 1px solid #F4F4F4;
}
#operatorPage .select-body-list i{
	visibility: hidden;
}
#operatorPage .select-body-list:hover i{
	visibility: visible;
}
#operatorPage .select-body-list:nth-child(2n+1){
	background: #FCFDFF;
}
#operatorPage  .el-input__suffix{
	right: 10px;
}
#operatorPage .el-pagination .el-input__suffix{
	right: 10px;
}
#operatorPage .queryGroup{
	padding-right: 5px!important;
}
#operatorPage .roleLifeCls{
	flex:1 1 20%;
	background:#FFF;
	font-size: 12px;
}
#operatorPage .roleRightCls{
	flex: 1 1 80%;
	background:#FFF;
	border-left:2px solid #e9e9e9;
	box-sizing:border-box;
	display: flex;
	flex-direction: column;
}
#operatorPage .roleLifeChild{
	display: flex;
	height: 36px;
	width: 100%;
	line-height: 36px;
	background-color: #FCFDFF;
	cursor: pointer;
}
#operatorPage .roleLifeChild div:first-child,#operatorPage .roleLifeChildSelect div:first-child{
	width: 40px;
	border-right: 1px solid #F3F3F3;
	text-align: center;
}
#operatorPage .roleLifeChild div:last-child,#operatorPage .roleLifeChildSelect div:last-child{
	padding-left: 5px;
}
#operatorPage .roleLifeChildSelect{
	display: flex;
	height: 36px;
	width: 100%;
	line-height: 36px;
	background-color: #EDF6FF;
	cursor: pointer;
}
#operatorPage .roleRightHeard{
	padding: 20px 0px 20px 40px;
	position:relative;
	display: flex;
}
#operatorPage .headContent{
	font-size:14px;
	color:#363B4E;
	font-weight:700;
	display:flex;
	align-items:center;
	padding-left: 40px;
	margin-bottom: 20px;
}
#operatorPage .headContent img{
	margin-right:10px;
}
#operatorPage .titleIconSty{
	position: relative;
	top: -5px;
	color: #FFF;
	background-color: #F3916C;
	height: 14px;
	width: 32px;
	border-radius: 10px;
	line-height: 14px;
	text-align: center;
	font-size: 12px;
}
#operatorPage .roleRightContent{
	height: 100%;
	flex: 1;
	overflow: auto;
}
#operatorPage .roleRightFooter{
	border-top: 1px solid #E9E9E9;
	height: 50px;
	padding-left: 40px;
	display: flex;
	align-items: center;
}
#operatorPage .alarmBottomLine{
	background-color:#E9E9E9;
	width: 100%;
	height: 1px;
	margin-bottom: 30px; 
}
#operatorPage .operatorListSty{
	height:400px;
	margin-left: 40px;
	display: flex;
	flex-direction: column;
	border:1px solid #DCDFE6
}
#operatorPage .betaIconClass{
	display: inline-block;
	position: relative;
	color: #FFF;
	top:-5px;
	left: 2px;
	background-color: #F3916C;
	height: 14px;
	width: 32px;
	border-radius: 10px;
	line-height: 14px;
	text-align: center;
	font-size: 10px;
}
#operatorPage .operatorNameClass{
	position: relative;
	display: flex;
	align-items: center;
}
#operatorPage .operatorNameClass .defaultClass{
	height: 8px;
	width: 8px;
	border-radius: 8px;
	background-color: #67C23A;
	margin-right: 5px;
}
.operatorNamePopoverClass{
	min-width: 80px!important;
}
#viewoperatorSlide .el-card__header{
	margin-left: 10px!important;
}
</style>
<div id="operatorPage"  style="overflow: auto;">
	<el-tabs v-model="operatorTypeTabs" @tab-click="operatorTypeTabsChange">
		<el-tab-pane label="<%=rb.getString("YunYingShang") %>" name="operator" style="overflow:auto">
			<div class="pageContainer">
				<!-- 运行商列表 -->
				<div class="operatorCon" style="overflow: auto;">
					<el-ctable ref="operListTable" @row-click="selectOperator" :url="operListTableUrl" :query-params="params" :height="height" pagination="true" :rownumber="rownumber">
						<template slot="toolbar">
							<div class="tableHeader">
								<span class="headerTitle"><%=rb.getString("YunYingShang") %></span>
								<div class="searchCon" >
									<el-input placeholder='<%=rb.getString("YunYingShangMingCheng") %>/ <%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("CPEMacAddress")%>'  @keyup.enter.native="refreshOperListTable" v-model="operListSearchText">
										<i slot="suffix" class="el-icon el-icon-common-search" style="margin-top:5px;" @click="refreshOperListTable"></i>
									</el-input>
								</div>
								<div v-if="false" class='buttonGroup' >
									<temp-button title='<%=rb.getString("TianJia") %>' :temp-class='importClass' @temp-click="dialogOperatorFormVisible = true"></temp-button>
								</div>
							</div>
							
							<!-- 添加运营商弹出框 -->
							<el-dialog title="New Operator" class="addOperDia" :close-on-click-modal=false :visible.sync="dialogOperatorFormVisible">
								<el-form ref="operatorAddForm" :model="form" :rules="rules" label-position="top">
									<el-form-item label="<%=rb.getString("YunYingShangMingCheng")%><%=rb.getString("MaoHao")%>" prop="operatorCode">
										<el-input :disabled="disableFlag" @blur="fillDefaultAdmin" v-model="form.operatorCode"></el-input>
									</el-form-item>
									<el-form-item label="<%=rb.getString("CLOUDKEY")%><%=rb.getString("MaoHao")%>" prop="cloudKey">
										<el-input :disabled="disableFlag" @focus="fillDefaultCloudKey" v-model="form.cloudKey"></el-input>
									</el-form-item>
									<el-form-item label="<%=rb.getString("MoRenGuanLiYuan")%><%=rb.getString("MaoHao")%>" prop="adminUserCode">
										<el-input :disabled="disableFlag" v-model="form.adminUserCode"></el-input>
									</el-form-item>
								</el-form>
								<div slot="footer" class="dialog-footer">
									<el-button type="primary" @click="addOperator"><%=rb.getString("QueDing")%></el-button>
									<el-button @click="dialogOperatorFormVisible = false"><%=rb.getString("QuXiao")%></el-button>
								</div>
							</el-dialog>
							
						</template>
						<el-table-column key="operation" prop="operation" label="" width="30"  class-name="operationColumn">
							<template slot-scope="scope">
								<template v-if="hideOperation">
									<div v-if="scope.row.isCurrent == '1'" class="disabled el-icon-operation-defaultOperator el-icon" title="<%=rb.getString("SheWeiDangQianYunYingShang") %>"></div>
									<div v-else class="el-icon-operation-defaultOperator el-icon" title="<%=rb.getString("SheWeiDangQianYunYingShang") %>" @click="setCurrentOperator(scope.row.operator_code)"></div>
								</template>
								<div v-if="!hideOperation" class="el-icon el-icon-operation-more" @click="optClick(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
							</template>
						</el-table-column>
						<el-table-column key="operator_name" prop="operator_name" width="160" >
							<template slot="header" slot-scope="scope">
								<span><%=rb.getString("YunYingShangMingCheng")%></span>
								<el-popover placement='bottom'  trigger='click' popper-class="operatorNamePopoverClass" v-model="isBeta">
									<el-checkbox v-model="params.is_beta" true-label="1" false-label="" @change="isBetaChange">Beta</el-checkbox>
									<span class="el-icon el-icon-common-filter" style="margin-left:10px;" slot='reference'></span>
								</el-popover>
							</template>
							<template slot-scope="scope">
								<div class="operatorNameClass"><div v-if="scope.row.isCurrent == '1'" class="defaultClass"></div>{{scope.row.operator_name}}<div class="betaIconClass" v-if="scope.row.is_beta == '1'">Beta</div></div>
							</template>
						</el-table-column>
						<el-table-column key="cloud_key" prop="cloud_key" label="<%=rb.getString("CLOUDKEY")%>" ></el-table-column>
						<el-table-column key="eNodeBCount" v-if="isEnbShow"  prop="eNodeBCount" label="<%=rb.getString("XiaoZhan")%>" ></el-table-column>
						<el-table-column key="CPECount" v-if="isCpeShow" prop="CPECount" label="<%=rb.getString("CPE")%>" ></el-table-column>
						<el-table-column key="gnbCount" v-if="isGnbShow" prop="gnbCount" label="gNB" ></el-table-column>
                        <el-table-column key="upsCount" v-if="isUpsShow" prop="upsCount" label="UPS" ></el-table-column>
						
					</el-ctable>
					<el-cmenu ref="operListMenu" @click="clickMenu" :data="menus"></el-cmenu>
					
				</div>
				<!-- eNB cpe gnb列表 -->
				<div class="operatTableList" style="overflow: auto;">
					<template>
						<el-tabs v-model="deviceTypeTabs" @tab-click="changeDeviceTable">
							<el-tab-pane label="eNB" name="enb" style="overflow:auto" v-if="isEnbShow">
								<el-ctable @selection-change="changeEnbSelect" ref="deviceEnbTable" :url="deviceEnbTableUrl" row-key="small_cell_code" :query-params="paramsEnb" :height="height" pagination="true" :rownumber="rownumber">
									<template slot="toolbar">
										<div class="searchCon" style="margin-left:18px;">
											<el-input placeholder="<%=rb.getString("XiaoZhanBianMa")%>" @keyup.enter.native="refreshEnbTable" v-model="enbSearchText">
												<i slot="suffix" class="el-icon el-icon-common-search" style="margin-top:5px;" @click="refreshEnbTable"></i>
											</el-input>
										</div>
									</template>
									<el-table-column type="selection" width="55" :reserve-selection="true"></el-table-column>
									<el-table-column prop="serial_number" label="<%=rb.getString("XiaoZhanBianMa")%>" ></el-table-column>
								</el-ctable>
							</el-tab-pane>
							<el-tab-pane label="CPE" name="cpe" style="overflow:auto" v-if="isCpeShow">
								<el-ctable @selection-change="changeCpeSelect" ref="deviceCpeTable" :url="deviceCpeTableUrl" row-key="cpe_code"  :query-params="paramsCpe" :height="height" pagination="true" :rownumber="rownumber">
									<template slot="toolbar">
										<div class="searchCon" style="margin-left:18px;">
											<el-input placeholder="<%=rb.getString("CPEXuLieHao")%>/<%=rb.getString("CPEMacAddress")%>" suffic-icon='el-icon-search' @keyup.enter.native="refreshCpeTable" v-model="cpeSearchText">
												<i slot="suffix" class="el-icon el-icon-common-search" style="margin-top:5px;"  @click="refreshCpeTable"></i>
											</el-input>
										</div>
									</template>
									<el-table-column type="selection" width="55" :reserve-selection="true"></el-table-column>
									<el-table-column prop="serial_number" label="<%=rb.getString("CPEXuLieHao")%>" ></el-table-column>
									<el-table-column prop="macaddress" label="<%=rb.getString("CPEMacAddress")%>" ></el-table-column>
								</el-ctable>
							</el-tab-pane>
							<el-tab-pane label="gNB" name="gnb" style="overflow:auto" v-if="isGnbShow">
								<el-ctable @selection-change="changeGnbSelect" ref="deviceGnbTable" :url="deviceGnbTableUrl" row-key="small_cell_code" :query-params="paramsGnb" :height="height" pagination="true" :rownumber="rownumber">
									<template slot="toolbar">
										<div class="searchCon" style="margin-left:18px;">
											<el-input placeholder="<%=rb.getString("XiaoZhanBianMa")%>" @keyup.enter.native="refreshGnbTable" v-model="gnbSearchText">
												<i slot="suffix" class="el-icon el-icon-common-search" style="margin-top:5px;" @click="refreshGnbTable"></i>
											</el-input>
										</div>
									</template>
									<el-table-column type="selection" width="55" :reserve-selection="true"></el-table-column>
									<el-table-column prop="serial_number" label="<%=rb.getString("XiaoZhanBianMa")%>" ></el-table-column>
								</el-ctable>
							</el-tab-pane>
                            <el-tab-pane label="UPS" name="ups" style="overflow:auto" v-if="isUpsShow">
								<el-ctable @selection-change="changeUpsSelect" ref="deviceUpsTable" :url="deviceUpsTableUrl" row-key="ups_code" :query-params="paramsUps" :height="height" pagination="true" :rownumber="rownumber">
									<template slot="toolbar">
										<div class="searchCon" style="margin-left:18px;">
											<el-input placeholder="<%=rb.getString("DianYuanBianMa")%>" @keyup.enter.native="refreshUpsTable" v-model="upsSearchText">
												<i slot="suffix" class="el-icon el-icon-common-search" style="margin-top:5px;" @click="refreshUpsTable"></i>
											</el-input>
										</div>
									</template>
									<el-table-column type="selection" width="55" :reserve-selection="true"></el-table-column>
									<el-table-column prop="serial_number" label="<%=rb.getString("DianYuanBianMa")%>" ></el-table-column>
								</el-ctable>
							</el-tab-pane>
						</el-tabs>
						<!-- @load-success="loadOperSuccess" --><!--:data="operatorSelectTableData" :url="operatorSelectTableUrl"-->
						<el-dialog title="<%=rb.getString("XuanZeYunYingShang")%>" :close-on-click-modal=false :visible.sync="dialogselectOperatorFormVisible" @close="cancelMoveTooperator" :append-to-body="true">
							<el-ctable border=true ref="changeOperaTable"  @row-click="rowClickUpgrade" :url="operatorSelectTableUrl" :query-params="paramsoper" :height="operheight" pagination="true" :rownumber="rownumber">
								<template slot="toolbar">
									<div class="searchCon" style="margin-left:0px;">
										<el-input placeholder="<%=rb.getString("YunYingShangMingCheng")%>" @keyup.enter.native="refreshSelectOper" v-model="operSearchText">
											<i slot="suffix" class="el-icon el-icon-common-search" style="margin-top:5px;" @click="refreshSelectOper"></i>
										</el-input>
									</div>
								</template>
								<el-table-column label='<%=rb.getString("XuanZe")%>' width="80">
								<template slot-scope="scope">
										<div class='tableDiv el-icon el-icon-status-yes selected-status' style="cursor: pointer;text-align:center;line-height:23px;"></div>
									</template>
								</el-table-column>
								<el-table-column prop="operator_name" label="<%=rb.getString("YunYingShangMingCheng")%>" ></el-table-column>
								<el-table-column prop="cloud_key" label="<%=rb.getString("CLOUDKEY")%>" ></el-table-column>
							</el-ctable>
							<div style="color:#F56C6C;height:20px;">
								<span v-if="operatorRowTag"><%=rb.getString("QingXuanZeYunYingShang")%></span>
							</div>
							<div slot="footer" class="dialog-footer">
								<el-button type="primary" @click="saveMoveTooperator"><%=rb.getString("QueDing")%></el-button>
								<el-button style='margin-left:10px;' @click="cancelMoveTooperator"><%=rb.getString("QuXiao")%></el-button>
							</div>
						</el-dialog>
					</template>
				</div>
			</div>
		</el-tab-pane>
		<el-tab-pane label="<%=rb.getString("JueSe") %>" name="role" style="overflow:auto" v-if="showRole">
			<div class="pageContainer">
				<!--左侧角色集列表-->
				<div class="roleLifeCls">
					<div :class="roleLifeVal == '1' ? 'roleLifeChildSelect' : 'roleLifeChild' " @click="roleLifeChildClick('1')">
						<div>1</div>
						<div>Admin-Role</div>
					</div>
					<div :class="roleLifeVal == '2' ? 'roleLifeChildSelect' : 'roleLifeChild' " @click="roleLifeChildClick('2')">
						<div>2</div>
						<div>Beta-Role</div>
						<div class="titleIconSty" style="top:5px;">Beta</div>
					</div>
				</div>
				<!--右侧角色集权限列表-->
				<div class="roleRightCls">
					<div class="roleRightContent">
						<div class="roleRightHeard" v-show="roleLifeVal == '1'"><div style="font-weight:550;">Admin-Role</div><%=rb.getString("GuanLiYuanJueSeTiShi") %></div>
						<div class="roleRightHeard" v-show="roleLifeVal == '2'">
							<div style="font-weight:550;">
								Beta-Role
							</div>
							<div class="titleIconSty">Beta</div>
						</div>
						<div class="headContent">
							<img src="${ctx}/css/images/global/settingBetter.png">
							<%=rb.getString("GongNengQuanXianLieBiao") %>
							<el-button style="margin-left:20px;" type="primary" v-if="!showViewFlag" @click="resetFeatureData"><%=rb.getString("QiYongMoRen")%></el-button>
						</div>
						<div style='margin:0px 0px 20px 68px;'>
							<el-ctree ref='featureTree' style="height: 500px;width: 80%;"
								@node-click="nodeClick"
								:readonly="showViewFlag"
								:default-expanded-keys="['0']"
								title="<%=rb.getString("QuanXianLieBiao")%>"
								placeholder='<%=rb.getString("QuanXianLieBiao")%>'
								:data="data1"
								:cascade="['wForms > forms']"
								:check-forms="[
									{key: 'forms',label: '<%=rb.getString("ZhiDuQuanXuan")%>',prop: 'checked'},
									{key: 'wForms',label: '<%=rb.getString("KeXieQuanXuan")%>',prop: 'write'}
								]"
								:ignore="featureIgnore"></el-ctree>
						</div>
						<div class="alarmBottomLine" v-show="false"></div>
						<div class="headContent" v-show="false">
							<img src="${ctx}/css/images/global/settingBetter.png"><%=rb.getString("YunYingShangLieBiao") %>
						</div>
						<div class="operatorListSty" style="margin:0px 40px 30px 68px;" v-show="false">
							<div class='queryGroup' style="margin:10px 20px;width:440px">
								<el-input v-model='queryOperatorListParams.search_text' @keyup.enter.native="queryOperatorList" class='pairgrid-query' placeholder='<%=rb.getString("YunYingShangMingCheng")%>'></el-input>
								<i @click='queryOperatorList' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
							</div>
							<el-ctable key="operatorKey" ref='operatorListTable' @load-success="tableLoadSuccess" @selection-change="changeOperatorList" :readonly="showViewFlag" :default-checked="defaultChecked" :url='operatorListUrl'  width='100%' height="100%"  :query-params="operatorListParams" pagination="true" row-key="operator_code">
								<el-table-column  type='selection' width='55'></el-table-column>
								<el-table-column prop='operator_name' label='<%=rb.getString("YunYingShangMingCheng")%>'></el-table-column>
								<el-table-column prop='cloud_key' label='CloudKey'></el-table-column>
							</el-ctable>
						</div>
					</div>
					<div class="roleRightFooter">
						<el-button type="primary" @click="showViewFlag = false" v-if="showViewFlag"><%=rb.getString("XiuGai")%></el-button>
						<el-button type="primary" v-if="!showViewFlag" @click="modifySubmit"><%=rb.getString("QueDing")%></el-button>
						<el-button  @click="modifyCancel" v-if="!showViewFlag"><%=rb.getString("QuXiao")%></el-button>
					</div>

				</div>
			</div>
		</el-tab-pane>
	</el-tabs>
	
	<!-- 设备选择 -->
	<div class="select-device-container" v-if="showflag">
		<div style="height:24px;"><span class="titleSpan" style="margin-left:10px;margin-right:10px;"><%=rb.getString("YiXuanSheBei")%></span>(<span style="color:#4D84FF">{{selectDeviceNum}}</span>)<i style="font-size:14px;margin-left:10px;" :class="showSelectDeviceFlag ? 'el-icon-circle-down' : 'el-icon-circle-up'" class="el-icon el-icon-circle-down" @click="showSelectDeviceFlag = !showSelectDeviceFlag"></i></div>
		<div class="">
			<el-button type="primary"  class="selfbutton" @click="moveOperator"><%=rb.getString("YiDongDaoYunYingShang")%></el-button>
			<el-button style="margin-right:0px;" class="selfbutton" class="selfbutton" @click="changeDeviceTable"><%=rb.getString("QuXiao")%></el-button>
		</div>
		
			<!-- 选择设备详情列表 -->
		<transition name="slide">	
			<div v-if="showSelectDeviceFlag" class="has-select-container">
				<div class="select-container-header">
					<span class="border-header-tltle">Selected Devices</span>		
					<i class="el-icon el-icon-close" @click="showSelectDeviceFlag = !showSelectDeviceFlag" style='position:absolute;right:10px;top:15px;'></i>		
				</div>
				<div class="select-container-body">
					<div class="select-body-header">
						<span v-if="deviceTypeTabs == 'enb' || deviceTypeTabs == 'gnb'" class="body-header-title"><%=rb.getString("XiaoZhanBianMa")%></span>
						<span v-if="deviceTypeTabs == 'cpe'" class="body-header-title"><%=rb.getString("MACDiZhi")%>+<%=rb.getString("CPEBianMa")%></span>
                        <span v-if="deviceTypeTabs == 'ups'" class="body-header-title"><%=rb.getString("DianYuanBianMa")%></span>
						<span><i class="el-icon el-icon-operation-delete" @click="changeDeviceTable"></i><span style="font-size:12px;color:#333333;font-family:PingFang SC;font-weight:bold;margin-left:2px;"><%=rb.getString("QingChu")%></span></span>
					</div>
					<div class="body-list" v-if="deviceTypeTabs == 'enb' || deviceTypeTabs == 'gnb'">
						<div v-for="(item,index) in hasSelectDevice" :key="index" class="select-body-list">
							<span>{{item.serial_number}}</span><i class="el-icon el-icon-circle-close" style='font-size:12px;' @click="cancelSelectDevide(item.small_cell_code)"></i>
						</div>
					</div>
					<div class="body-list" v-if="deviceTypeTabs == 'cpe'">
						<div v-for="(item,index) in hasSelectDevice" :key="index" class="select-body-list">
							<span>{{item.macaddress}}({{item.serial_number}})</span><i class="el-icon el-icon-circle-close" style='font-size:12px;' @click="cancelSelectDevide(item.cpe_code)"></i>
						</div>
					</div>
                    <div class="body-list" v-if="deviceTypeTabs == 'ups'">
						<div v-for="(item,index) in hasSelectDevice" :key="index" class="select-body-list">
							<span>{{item.serial_number}}</span><i class="el-icon el-icon-circle-close" style='font-size:12px;' @click="cancelSelectDevide(item.ups_code)"></i>
						</div>
					</div>
				</div>
			</div>
		</transition>
	</div>
	
	<el-slide ref="viewoperatorSlide" id="viewoperatorSlide" :url='viewSlideUrl' :title="slideTitle" :footer="slideFooter" :header="slideHeader" :position="slidePosition" class="tableContent"
	:height="slideHeight" :modal='modal' :width='slideWidth' @ok="saveSlide" @cancel="cancelSlide" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
	</el-slide>
	<!-- 删除运营商弹出框 -->
	<el-dialog class="deldialog" title="Confirm" width="25%" :close-on-click-modal=false :visible.sync="dialogdelFormVisible" :append-to-body="true">
		<span><%=rb.getString("QueRenShanChuYunYingShang")%></span>
		<el-form ref="delForm" :model="delForm" :rules="delRules">
			<el-form-item label="Password" prop="delPassword">
				<el-input type="password" v-model="delForm.delPassword" auto-complete="new-password" placeholder="<%=rb.getString("QingShuRuMiMa")%>"></el-input>
			</el-form-item>
				<el-input type="password" v-show=false placeholder="<%=rb.getString("QingShuRuMiMa")%>"></el-input>
		</el-form>
			<div slot="footer" class="dialog-footer" style="display:unset;">
				<el-button type="primary" @click="suerDel"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="dialogdelFormVisible = false"><%=rb.getString("QuXiao")%></el-button>
			</div>
	</el-dialog>
	
</div>

<template id='elCardTopButton'>
	<div>
		<i :class='tempClass' @click="$emit('temp-click')" @mouseover.native='showText' @mouseout.native='hideText'></i>
		<p v-show='operText'>{{title}}</p>
	</div>
</template>
<script type="text/javascript">
	new Vue({
		el:'#operatorPage',
		data(){
			var validateDefaultAdmin = (rule,value,callback)=>{
				var reg = /^[a-zA-Z0-9\_\-\@\.]+$/;
				//必填验证
				if(!value){
					callback(new Error('<%=rb.getString("QingShuRuYunYingShangMingCheng")%>'));
				}else{
					if(reg.test(value)){
						callback()
					}else{
						callback(new Error("<%=rb.getString("YunYingShangChengBuHeFa")%>"));
					}
				}
			}
			var validateCloudKey = (rule,value,callback)=> {
				var reg = /^[0-9A-Z]{6}$/;
				if(reg.test(value)){
					callback();
				}else{
					callback(new Error("<%=rb.getString("CLOUDKEYYOUXIAOXING")%>"));
				}
			}
			return {
				roleLifeVal:'1',
				operatorTypeTabs:'operator',
				showSelectDeviceFlag:false,
				hasSelectDevice:[],
				delOperatorCode:'',
				viewSlideUrl:'',
				slideTitle:'',
				slideFooter:'',
				slideHeader:'',
				slidePosition:'',
				slideHeight:'',
				slideWidth:'',
				modal:'',
				showflag:false,
				selectDeviceNum:0,
				deviceTypeTabs:'enb',
				disableFlag:false,
				enbSearchText:'',
				gnbSearchText:'',
				cpeSearchText:'',
                upsSearchText:'',
				operSearchText:'',
				form:{
					operatorCode:'',
					cloudKey:'',
					operatorName:'',
					adminUserCode:''
				},
				delForm:{
					delPassword:'',
				},
				dialogOperatorFormVisible:false,
				dialogselectOperatorFormVisible:false,
				dialogdelFormVisible:false,
				searchText:'',
				operListSearchText:'',
				params:{
					searchText:'',
					is_beta:'',
				},
				isBeta:false,
				operator_code_current:operator_code,
				paramsEnb:{
					search_text:'',
					operator_code:operator_code,
					like_fields:'serial_number'
				},
				paramsGnb:{
					search_text:'',
					isGnb:1,
					operator_code:operator_code,
					like_fields:'serial_number'
				},
				paramsCpe:{
					search_text:'',	
					operator_code:operator_code,
					like_fields:'serial_number,macaddress'
				},
                paramsUps:{
					searchText:'',
					operator_code:operator_code,
					like_fields:'serial_number'
				},
				paramsoper:{
					operator_code:'',
					operator_code_current:operator_code
					//like_fields:'serial_number'
				},
				importClass:'el-icon-circle-add el-icon',
				operListTableUrl:'${ctx}/system/operator/getOperatorListAndCounts.action',
				deviceEnbTableUrl:'${ctx}/system/device/enodeb/queryENBInfoPageList.action',
				deviceGnbTableUrl:'${ctx}/system/device/enodeb/queryENBInfoPageList.action',
				deviceCpeTableUrl:'${ctx}/system/device/cpe/queryCPEInfoPageList.action',
                deviceUpsTableUrl:'${ctx}/ups/queryUpsInfosList.action',
				operatorSelectTableUrl:'${ctx}/system/operator/getOperatorListExceptself.action?no_built_in=1',
				
				height:'100%',
				operheight:'40%',
				pageSize:50,
				rownumber:true,
				menus:[],
				enbIDs:'',
				gnbIDs:'',
				cpeCodes:'',
                upsCodes:'',
				operatorRow:'',
				delRules:{
					delPassword:{required:true,message:"<%=rb.getString("QingShuRuMiMa")%>",trigger:'blur'}
				},
				rules:{
					operatorCode:[
						{validator:validateDefaultAdmin}
					],
					cloudKey:[
						{validator:validateCloudKey}
					],
					adminUserCode:[
						{require:true,message:'<%=rb.getString("QingShuRuYunYingShangMingCheng")%>'}
					]
				},
				hideOperation:isCloudCore == "true" ? true : false,
				role_id:'1',
				showViewFlag:true,
				featureIgnore: {
					forms: ['1','7','10','30','32','35','38','64','76','92','96','97','98','122'],
					wForms: ['1','58','59','60','62','63','117']
				},
				data1:'',
				result: {},
				resultDefault:{
					readonly:[]
				},
				defaultChecked:[],
				queryOperatorListParams:{
					search_text:''
				},
				operatorListParams:{
					operator_code:'',
					role_id:'2',
					type:'modify'
				},
				operatorListUrl:'',
				operatorCodeAll:'',
				operatorCodeNoAll:'',
				intSelect:[],  // 运营商初始选中
				operatorRowTag:false,
				setOperatorFlag:false,
				idRelCodeMap: {}
			}
		},
		computed: {
			showRole(){
				return isDistributor != '1'
			},
			isGnbShow(){
				return  writableMap['CODE_GNB_MONITOR'] !== undefined;
			},
			isCpeShow(){
				return  writableMap['CODE_CPE_MONITOR'] !== undefined;
			},
			isEnbShow(){
				return  writableMap['CODE_ENB_MONITOR'] !== undefined;
			},
            isUpsShow(){
				return  writableMap['CODE_UPS'] !== undefined;
			},
		},
		methods:{
			//角色 右侧点击事件
			roleLifeChildClick(val){
				var vm = this;
				vm.roleLifeVal = val;
				vm.$refs.featureTree.reset();
				vm.$refs.featureTree.resetQuery();
				//vm.$refs.operatorListTable.refresh();
				setTimeout(()=>{
					vm.getFeatureData();
					vm.showViewFlag = true;
				},0)
				
			},
			// 左侧运营商列表单行点击事件
			selectOperator(row,column,event){
				var vm = this;

				if(vm.operator_code_current != row.operator_code){
					vm.paramsCpe.operator_code = row.operator_code;
					vm.paramsEnb.operator_code = row.operator_code;
					vm.paramsGnb.operator_code = row.operator_code;
                    vm.paramsUps.operator_code = row.operator_code;
					vm.paramsoper.operator_code_current = row.operator_code;
					vm.operator_code_current = row.operator_code;
					vm.refreshEnbTable();
					vm.refreshGnbTable();
					vm.refreshCpeTable();
                    vm.refreshUpsTable();
					
					if(vm.deviceTypeTabs == "enb"){
						vm.$refs["deviceEnbTable"].clearSelection();
					}else if(vm.deviceTypeTabs == "gnb"){
						vm.$refs["deviceGnbTable"].clearSelection();
					}else if(vm.deviceTypeTabs == "cpe"){
						vm.$refs["deviceCpeTable"].clearSelection();
					}else if(vm.deviceTypeTabs == "ups"){
						vm.$refs["deviceUpsTable"].clearSelection();
					}
				}
				
			},
			// enb 列表选择事件
			changeEnbSelect(selection){
				var vm = this;
				vm.hasSelectDevice = [];
				vm.hasSelectDevice = selection;
				vm.selectDeviceNum = selection.length;
				if(vm.selectDeviceNum != 0){
					vm.showflag = true;
				}else{
					this.showSelectDeviceFlag = false;
					vm.showflag = false;
				}
				var ids = [];
				vm.hasSelectDevice.map((item)=>{
					var targetVal = '';
					targetVal = item.small_cell_code + "_" + item.product;
					ids.push(targetVal);
				})
				vm.enbIDs = ids.join(",");
			},
			// Gnb 列表选择事件
			changeGnbSelect(selection){
				var vm = this;
				vm.hasSelectDevice = [];
				vm.hasSelectDevice = selection;
				vm.selectDeviceNum = selection.length;
				if(vm.selectDeviceNum != 0){
					vm.showflag = true;
				}else{
					this.showSelectDeviceFlag = false;
					vm.showflag = false;

				}
				var ids = [];
				vm.hasSelectDevice.map((item)=>{
					var targetVal = '';
					targetVal = item.small_cell_code + "_" + item.product;
					ids.push(targetVal);
				})
				vm.gnbIDs = ids.join(",");
			},
			// cpe 列表选择事件
			changeCpeSelect(selection){
				var vm = this;
				vm.hasSelectDevice = [];
				vm.hasSelectDevice = selection;
				vm.selectDeviceNum = selection.length;
				if(vm.selectDeviceNum != 0){
					vm.showflag = true;
				}else{
					this.showSelectDeviceFlag = false;
					vm.showflag = false;
					
				}
				var cpeCodes = [];
				vm.hasSelectDevice.map((item)=>{
					var targetVal = '';
					targetVal = item.cpe_code;
					cpeCodes.push(targetVal);
				})
				vm.cpeCodes = cpeCodes.join(",");
			},
            // ups 列表选择事件
			changeUpsSelect(selection){
				var vm = this;
				vm.hasSelectDevice = [];
				vm.hasSelectDevice = selection;
				vm.selectDeviceNum = selection.length;
				if(vm.selectDeviceNum != 0){
					vm.showflag = true;
				}else{
					this.showSelectDeviceFlag = false;
					vm.showflag = false;
					
				}
				var upsCodes = [];
				vm.hasSelectDevice.map((item)=>{
					var targetVal = '';
					targetVal = item.ups_code;
					upsCodes.push(targetVal);
				})
				vm.upsCodes = upsCodes.join(",");
			},
			changeDeviceTable(){
				//清空选择项
				var vm = this;
				this.showSelectDeviceFlag = false;
				vm.showflag = false;
				if(this.deviceTypeTabs == "enb"){
					vm.$refs["deviceEnbTable"].clearSelection();
				}else if(this.deviceTypeTabs == "gnb"){
					vm.$refs["deviceGnbTable"].clearSelection();
				}else if(this.deviceTypeTabs == "cpe"){
					vm.$refs["deviceCpeTable"].clearSelection();
				}else if(this.deviceTypeTabs == "ups"){
					vm.$refs["deviceUpsTable"].clearSelection();
				}
				
			},
			// 单个删除已选设备
			cancelSelectDevide(smallCellCode){
				var vm = this;
				let tableData = null;
				let rowIndex = 0;
				if(this.deviceTypeTabs == "enb"){
					tableData = vm.$refs["deviceEnbTable"].getData();
					tableData.some((item,index)=>{
						if(item.small_cell_code == smallCellCode){
							return (rowIndex = index);
						}
					})
				}else if(this.deviceTypeTabs == "gnb"){
					tableData = vm.$refs["deviceGnbTable"].getData();
					tableData.some((item,index)=>{
						if(item.small_cell_code == smallCellCode){
							return (rowIndex = index);
						}
					})
				}else if(this.deviceTypeTabs == "cpe"){
					tableData = vm.$refs["deviceCpeTable"].getData();
					tableData.some((item,index)=>{
						if(item.cpe_code == smallCellCode){
							return (rowIndex = index);
						}
					})
				}else if(this.deviceTypeTabs == "ups"){
					tableData = vm.$refs["deviceUpsTable"].getData();
					tableData.some((item,index)=>{
						if(item.ups_code == smallCellCode){
							return (rowIndex = index);
						}
					})
				}
				if(this.deviceTypeTabs == "enb"){
					tableData = vm.$refs["deviceEnbTable"].toggleRowSelection(tableData[rowIndex],false);
				}else if(this.deviceTypeTabs == "gnb"){
					tableData = vm.$refs["deviceGnbTable"].toggleRowSelection(tableData[rowIndex],false);
				}else if(this.deviceTypeTabs == "cpe"){
					tableData = vm.$refs["deviceCpeTable"].toggleRowSelection(tableData[rowIndex],false);
				}else if(this.deviceTypeTabs == "ups"){
					tableData = vm.$refs["deviceUpsTable"].toggleRowSelection(tableData[rowIndex],false);
				}
			},
			// 打开移动到运营商弹窗
			moveOperator(){
				var vm = this;
				if(vm.deviceTypeTabs == "enb"){
					if(vm.enbIDs == ""){
						vm.$message('<%=rb.getString("QingXuanZeSheBei")%>')
					}
				}else if(vm.deviceTypeTabs == "gnb"){
					if(vm.gnbIDs == ""){
						vm.$message('<%=rb.getString("QingXuanZeSheBei")%>')
					}
				}else if(vm.deviceTypeTabs == "cpe"){
					if(vm.cpeCodes == ""){
						vm.$message('<%=rb.getString("QingXuanZeSheBei")%>')
					}
				}else if(vm.deviceTypeTabs == "ups"){
					if(vm.upsCodes == ""){
						vm.$message('<%=rb.getString("QingXuanZeSheBei")%>')
					}
				}
				//vm.operSearchText = ''; //清空搜索项
				//vm.paramsoper.operator_code = ''; //搜索参数清空
				vm.dialogselectOperatorFormVisible = true;
			},
			// select operator 确定事件
			saveMoveTooperator(){
				var vm = this;
				//判断是否选择了
				if(!this.operatorRow){
					// vm.$message.error("<%=rb.getString("QingXuanZeYunYingShang")%>") //错误提示信息
					vm.operatorRowTag = true
					return false;
				}
				var params = {};
				var url = ""
				if(this.deviceTypeTabs == "enb"){
					params.ids = this.enbIDs;
					url="${ctx}/system/deviceGroup/moveCellToOperator.action";
				}else if(this.deviceTypeTabs == "gnb"){
					params.ids = this.gnbIDs;
					params.isGnb = 1;
					url="${ctx}/system/deviceGroup/moveCellToOperator.action";
				}else if(this.deviceTypeTabs == "cpe"){
					params.cpeCodes = this.cpeCodes;
					url="${ctx}/cell/CPE/moveCpeToOperator.action";
				}else if(this.deviceTypeTabs == "ups"){
					params.ids = this.upsCodes;
					url="${ctx}/system/deviceGroup/moveUpsToOperator.action";
				}
				params.toOperatorCode = this.operatorRow.operator_code;
				axios.post(url,stringify(params)).then(function(response){
		    		var data = response.data;
		    		if(data["success"]){
						this.showSelectDeviceFlag = false;
		    			vm.showflag = false;
		    			if(vm.deviceTypeTabs == "enb"){
		    				vm.$refs["deviceEnbTable"].refresh();
							vm.$refs["deviceEnbTable"].clearSelection();
		    			}else if(vm.deviceTypeTabs == "gnb"){
		    				vm.$refs["deviceGnbTable"].refresh();
							vm.$refs["deviceGnbTable"].clearSelection();
		    			}else if(vm.deviceTypeTabs == "cpe"){
		    				vm.$refs["deviceCpeTable"].refresh();
							vm.$refs["deviceCpeTable"].clearSelection();
		    			}else if(vm.deviceTypeTabs == "ups"){
		    				vm.$refs["deviceUpsTable"].refresh();
							vm.$refs["deviceUpsTable"].clearSelection();
		    			}
		    			vm.$refs["operListTable"].refresh();
		    		
		    			vm.dialogselectOperatorFormVisible = false;
		    			vm.$refs.changeOperaTable.refresh();
		    			vm.$refs.changeOperaTable.setCurrentRow();
		    			vm.operatorRow = '';
		    			vm.operSearchText = ''; //清空搜索项
						vm.paramsoper.operator_code = ''; //搜索参数清空
		    			vm.$message({
		    				type:'success',
		    				message:"<%=rb.getString("ChengGong")%>"
		    			})
		    		}else{
		    			vm.$message.error(data["message"]) //错误提示信息
		    		}
		    	})
			},
			//select operator 取消事件  取消被选中的数据
			cancelMoveTooperator(){
				this.dialogselectOperatorFormVisible = false;
    			this.$refs.changeOperaTable.refresh();
    			this.operatorRow = '';
    			this.operSearchText = ''; //清空搜索项
    			this.paramsoper.operator_code = ''; //搜索参数清空
    			this.$refs.changeOperaTable.setCurrentRow();
			},
			refreshOperListTable(){
				var vm = this;
				vm.params.searchText = vm.operListSearchText;
				vm.$refs["operListTable"].refresh();
			},
			refreshEnbTable(){
				var vm = this;
				vm.paramsEnb.search_text = vm.enbSearchText;
				vm.$refs["deviceEnbTable"].refresh();
			},
			refreshGnbTable(){
				var vm = this;
				vm.paramsGnb.search_text = vm.gnbSearchText;
				vm.$refs["deviceGnbTable"].refresh();
			},
			refreshCpeTable(){
				var vm = this;
				vm.paramsCpe.search_text = vm.cpeSearchText;
				vm.$refs["deviceCpeTable"].refresh();
			},
            refreshUpsTable(){
				var vm = this;
				vm.paramsUps.searchText = vm.upsSearchText;
				vm.$refs["deviceUpsTable"].refresh();
			},
			refreshSelectOper(){
				var vm = this;
				vm.paramsoper.operator_code = vm.operSearchText;
				vm.paramsoper.operator_code_current = vm.operator_code_current
				vm.$refs["changeOperaTable"].refresh();
			},
			suerDel(){
				var vm = this;
				var params = {"operatorCode": this.delOperatorCode,adminPassword:this.delForm.delPassword};
				this.$refs["delForm"].validate((valid)=>{
					 if(valid){
						 axios.post('${ctx}/system/operator/delOperator.action',stringify(params)).then(function(response){
					    		var data = response.data;
					    		if(data["success"]){
					    		
					    			vm.$refs["operListTable"].refresh();
					    			if(vm.delOperatorCode == operator_code){
					    				setCurrOperator(operator_code);
					    			}
					    			vm.$refs["delForm"].resetFields();
					    			vm.dialogdelFormVisible = false;
					    		}else{
					    			vm.$message.error("<%=rb.getString("ShanChuYunYingShangShiBai")%>") //错误提示信息
					    		}
					    	})
					 }
				})
			},
			
			optClick(row,ev){
		    	var vm = this,betaFlag=false,operatorFlag=false;
		    	this.rowData = row;
		    	var disableFlag = null;
		    	if(row.built_in != '1'){
		    		disableFlag = false;
		    	} else{
		    		disableFlag = true;
		    	}
				if(row.isCurrent == '1'){
					operatorFlag = true;
				}
				if(row.is_beta == '1'){
					betaFlag = true;
				}
				if(vm.hideOperation){
					vm.menus = [
						{label:'<%=rb.getString("SheWeiDangQianYunYingShang")%>',cls:'el-icon-operation-defaultOperator el-icon',code:'setOperator',disable:operatorFlag},
					]
				}else{
					vm.menus = [
						{label:'<%=rb.getString("XinXi")%>',cls:'el-icon-operation-info el-icon',code:'info'},
						{label:'<%=rb.getString("XiuGai")%>',cls:'el-icon-operation-edit el-icon',code:'edit'},
						{label:'<%=rb.getString("SheWeiDangQianYunYingShang")%>',cls:'el-icon-operation-defaultOperator el-icon',code:'setOperator',disable:operatorFlag},
						{label:'<%=rb.getString("SheWeiBeta")%>',cls:'el-icon-operation-defaultBeta el-icon',code:'setBeta',disable:betaFlag},
						{label:'<%=rb.getString("ShanChu")%>',cls:'el-icon-operation-delete el-icon',code:'del',disable:disableFlag}
					]
				}
		    	
		    	this.$nextTick(()=>{
					document.body.click();
					vm.$refs.operListMenu.show(ev)
				})
		    },
		    clickMenu(ev){ //菜单点击对应的方法
				var codes = {
					info:this.infoOperList,
					del:this.delOperList,
					edit:this.editOperList,
					setOperator:this.setCurrentOperator,
					setBeta:this.setBeta
				};
				//根据code 判断执行哪个方法
				codes[ev.code](this.$root.rowData["operator_code"]);
		  	},
			// 打开运营商详情弹窗
			infoOperList(operatorCode){
				var vm = this;
				vm.slideHeader = false;
				vm.slideTitle = '<%=rb.getString("XinXi")%>';
				vm.viewSlideUrl = '${ctx}/system/operator/toView.action';
				vm.slideFooter = false;
				vm.slidePosition = 'right';
				vm.slideHeight = '100%';
				vm.slideWidth = '65%';
				vm.$refs.viewoperatorSlide.showSlide(()=>{
					vm.model = true;
					eventBus.$emit('info-task',operatorCode,'view')
				})
			},
			// 打开运营商修改弹窗
			editOperList(operatorCode){
				var vm = this;
				vm.slideHeader = false;
				vm.slideTitle = '<%=rb.getString("XiuGai")%>';
				vm.viewSlideUrl = '${ctx}/system/operator/toView.action';
				vm.slideFooter = true;
				vm.slidePosition = 'right';
				vm.slideHeight = '100%';
				vm.slideWidth = '65%';
				vm.$refs.viewoperatorSlide.showSlide(()=>{
					vm.model = true;
					eventBus.$emit('info-task',operatorCode,'modify')
				})
			},
			saveSlide(){
				eventBus.$emit('save-edit-task')
			},
			cancelSlide(){
				var vm = this;
				vm.$refs.viewoperatorSlide.hide();
				vm.$refs["operListTable"].refresh();
			},
			// 打开运营商删除弹窗
			delOperList(operatorCode){
				this.dialogdelFormVisible = true;
				this.delOperatorCode = operatorCode;
				eventBus.$emit("del-operator",operatorCode)
			},
			// 设置当前运营商
			setCurrentOperator(operatorCode){
				var vm = this,
					params={
						operator_code:operatorCode
					};
				
					setCurrOperator(operatorCode);
					closeDefaultWindow();
					if("${omcVersion}" == "1"){//移动版本，如果在测量定制页面切换到普通运营商，则跳转到性能查询页面
						ChangeLanguage("<%=languageCode%>");
						if($("#customProgressbar").length != 0){
							$('#mainpage').panel({
								border: false,
								href: '${ctx}/cell/perfmgmt/goPerfStatisQuery.action'
							});
						}else{
							$('#mainpage').panel('refresh');
						}
					}else{
						try{
							$('#mainpage').panel('refresh');
						}catch(e){}
						ChangeLanguage("<%=languageCode%>");
					}
				
			},
			// 设为Beta
			setBeta(operatorCode){
				var vm = this,
					params={
						operator_code:operatorCode
					};
				var confirmStr = '<%=rb.getString("SheWeiBetaTiShi")%>';
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					closeOnClickModal:false
				}).then(() => {
					vm.$message.success('<%=rb.getString("ChengGong")%>')
					axios.post('${ctx}/system/operator/switchOperatorBeta.action',stringify(params)).then(function(response){
			    		var data = response.data;
			    		if(data["success"]){
			    			vm.$message.success('<%=rb.getString("ChengGong")%>')
			    			vm.$refs.operListTable.refresh();//表格刷新
			    		}else{
			    			vm.$message.error('<%=rb.getString("ShiBai")%>') //错误提示信息 
			    		}
			    	}) 
				}).catch(() => {
					
				})
			},
			rowClickUpgrade(row){
				this.operatorRow = row;
				if(this.operatorRow){
					this.operatorRowTag = false;
				}
			},
			handerClose(){
				this.$refs.operListMenu.hide();
			},
			//添加运营商相关的函数
			//随机生成CLOUDKEY
			fillDefaultCloudKey(){
				if(this.form.cloudKey == ""){
					this.form.cloudKey = Math.random().toString(30).substring(5).slice(0,6).toUpperCase();
				}
			},
			fillDefaultAdmin(){
				this.form.adminUserCode = this.form.operatorCode + "Admin"
			},
			operatorTypeTabsChange(val){
				var vm = this;
				if(vm.operatorTypeTabs == 'role'){
					vm.getFeatureData();
					vm.changeDeviceTable();
				}

			},
			// 权限树结构数据请求
			getFeatureData(){
				var vm= this,
					params = {
						type:'modify',
						role_id: vm.roleLifeVal,
					};
				axios.post("${ctx}/system/superRoleSet/getFeature.action",stringify(params)).then(function(response){
					var data = vm.handleData(response.data[0].children);
					// 初始不显示的节点
					// vm.initIgnore(data);
					vm.$nextTick(function(){
						// 设置默认勾选
						setTimeout(function(){
							vm.getResult();
							vm.resultDefault = vm.result;
						},100)
						
						vm.data1 = JSON.parse(JSON.stringify(data))
						setTimeout(function(){
							vm.$refs.featureTree.reviewForms(vm.data1);
						},0)
					
					});
				})
			},
			// 权限树结构数据请求
			resetFeatureData(){
				var vm= this,
					params = {
						type:'reset',
						reset:'YES',
						role_id: vm.roleLifeVal,
					};
				
				axios.post("${ctx}/system/superRoleSet/getFeature.action",stringify(params)).then(function(response){
					var data = vm.handleData(response.data[0].children);
					// 初始不显示的节点
					// vm.initIgnore(data);
					vm.$refs.featureTree.reset();
					vm.$nextTick(function(){
						vm.data1 = JSON.parse(JSON.stringify(data))
						setTimeout(function(){
							vm.$refs.featureTree.reviewForms(vm.data1);
						},0)
					
					});
				})
			},
			// 添加运营商 确定
			addOperator(){
				var vm = this;
				this.$refs["operatorAddForm"].validate((valid)=>{
					if(valid){
						var params = {};
						Object.assign(params,vm.form);
						axios.post('${ctx}/system/operator/addOperator.action',stringify(params)).then(function(response){
							var data = response.data;
							if(data["success"]){
								vm.dialogOperatorFormVisible = false;
								vm.$refs["operListTable"].refresh();
								vm.$refs["operatorAddForm"].resetFields();
							}else{
								vm.$message.error(data["message"]) //错误提示信息
							}
						})
					}
				})
			},
			// 角色 权限点击事件
			nodeClick(obj,node,t) {
				var vm = this;
				
				vm.$nextTick(function(){
					vm.getResult();
				});
			},
			//获取权限勾选结果
			getResult(){
				var vm = this,
					res = vm.$refs.featureTree.getResult(),
					rlist = res.forms,
					wlist = res.wForms;
				
				vm.result = {
					readonly: rlist.sort(function(a,b){return a-b}),
					write: wlist.sort(function(a,b){return a-b})
				};
			},
			/**
			 * 处理数据
			 * param data {array} 生成树的节点数据
			         type {string} 类型
			*/
			handleData(data){
				var vm = this;
				data.map(function(item){
					
					if(item.pid == '0'){
						item.pid = 'root';
					}
					if(item.children.length >0){
						vm.handleData(item.children)
						item.isLeaf = false
					}else{
						item.isLeaf = true;
					}
				})
				return data;
			},
			// 初始化不显示的节点
			initIgnore(data) {
				var vm = this; //featureIgnore
				data.map(function(row){
					if(row.children && row.children.length) vm.initIgnore(row.children);
					
					if(row.reWrite == 'true' || row.reWrite == true) {
						
					}else {
						row.isLeaf && vm.featureIgnore.wForms.push(row.id);
					}
				})
			},
			// 运营商列表选择事件
			changeOperatorList(selection){
				var vm = this,selectionData = selection,operatorCodeNoAll=[];
				if(selectionData.length != 0){
					selectionData.map((item)=>{
						operatorCodeNoAll.push(item.operator_code)
					})
				}
				vm.operatorCodeNoAll = operatorCodeNoAll;
			},
			//运营商列表模糊查询
			queryOperatorList(){
				var vm = this;
				vm.operatorListParams.operator_code = vm.queryOperatorListParams.search_text;
			},
			// 角色集修改提交
			modifySubmit(){
				var vm = this;
				var changeFlag = vm.checkChange();
				if(changeFlag == 'true'){
					vm.getResult();
					
					vm.idRelCodeMap = {};
					vm.getIdMap(vm.data1);
					
					var menu_ids = [{id: '1',code: 'CODE_DASHBOARD',write: true}];
					vm.result.write.map(function(item){ 
						var code = vm.idRelCodeMap[item];
						menu_ids.push({id:item,code: code,write:true});
					})
					vm.result.readonly.map(function(item){ 
						var code = vm.idRelCodeMap[item];
						if(!vm.result.write.includes(item)){
							menu_ids.push({id:item,code: code,write:false})
						}
					})
					
					var params = {
						role_id:vm.roleLifeVal*1,
						menu_ids:menu_ids,
					}
					/* if(vm.roleLifeVal == '1'){
						params.operator_codes = vm.operatorCodeNoAll
					}else{
						params.operator_codes = vm.intSelect
						
					} */
					params = JSON.stringify(params);
					$.post("${ctx}/system/superRoleSet/save.action",{"params":params},function(data){
						var message = '<%=rb.getString("ChengGong")%>';
						if(data["success"]){
							vm.showViewFlag = true;
							vm.$message({
								message:message,
								type:'success',
							})
						}else{
							vm.$message.error(data["message"]) //错误提示信息
						}
					},'json')
				}else{
					vm.$message("<%=rb.getString("WuCanShuBianHua")%>")
				}
				
			},
			getIdMap(list) {
				var vm = this;
				
				list.map(function(item) {
					var sub = item.children;
					
					if(sub && sub.length) {
						vm.getIdMap(sub);
					}
					
					vm.idRelCodeMap[item.id] = item.code;
				})
			},
			//校验参数是否有变化
			checkChange(){
				var vm = this;
				vm.getResult();
				var newSelect = vm.$refs.operatorListTable.getChecked();
				var readChange = vm.result.readonly.toString() != vm.resultDefault.readonly.toString(); //有改变
				var writeChange = vm.result.write.toString()  != vm.resultDefault.write.toString()//有改变
				var operListChange = stringify(newSelect.sort(function(a,b){return a-b})) !=  stringify(vm.intSelect.sort(function(a,b){return a-b}))
				if(newSelect.length == 0 && vm.intSelect.length == 0){
					operListChange = false;
				}
				var treeChange = readChange || writeChange || operListChange;
				if(treeChange){
					return "true";
				}else{
					return "false";
				}
			},
			// 运营商列表  表格加载成功回调
			tableLoadSuccess(){
				var vm = this,
					operatorCodeAll=[],
					intSelect=[];
				vm.$refs.operatorListTable.clearSelection();
				var operatorListData = vm.$refs.operatorListTable.getData();
				operatorListData.map(function(row){
					operatorCodeAll.push(row.operator_code);
					if(row.check == 'true'){
						intSelect.push(row.operator_code);
						vm.$nextTick(()=>{
							vm.$refs.operatorListTable.toggleRowSelection(row, true);
						})
					}
				})
				vm.operatorCodeAll = operatorCodeAll;
				vm.intSelect = intSelect;
			},
			//修改取消
			modifyCancel(){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>';
				var changeFlag = vm.checkChange();
				if(changeFlag == "true"){
					vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(() => {
						vm.$refs.featureTree.reset();
						/* if(vm.roleLifeVal == '1'){
							vm.$refs.operatorListTable.refresh();
						} */
						setTimeout(()=>{
							vm.getFeatureData();
							vm.showViewFlag = true;
						},0)
					}).catch(() => {
						
					})
				}else{
					
					vm.showViewFlag = true
				}
			},
			isBetaChange(){
				var vm = this;
				vm.isBeta = false;
			},
		},
		mounted(){
			eventBus.$off("close-slide").$on("close-slide",this.cancelSlide);
		},
		components:{
			'temp-button':{
				template:'#elCardTopButton',
				data(){
					return{
						operText:false
					}
				},
				props:['title','tempClass'],
				methods:{
					showText(){
						this.operText = true;
					},
					hideText(){
						this.operText = false;
					},
					
				}
			},
		},
	
	})
</script>