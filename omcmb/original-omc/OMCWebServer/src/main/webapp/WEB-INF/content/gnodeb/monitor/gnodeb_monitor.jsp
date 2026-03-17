<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>
<html>
	<head>
		<script type="text/javascript" src="${ctx}/js/element/Sortable.min.js?_=${omc_ver}"></script>
		<script type="text/javascript" src="${ctx}/js/element/vuedraggable.umd.min.js?_=${omc_ver}"></script>
		<style type="text/css">
			.cellActivePopoverClass{
				padding: 10px;
			}
			.no-padding .slide-content {
				padding: 0px;
			}
			.commonSlideClass {
				width: 72% !important;
				right: 0 !important;
				left: unset !important;
			}
			/* 列表告警显示样式 */
			#gnodeb_ctn .alarmListSty{
				display:inline-block;
				width:23px;
				height:23px;
				line-height:24px;
				border-radius:23px;
				text-align:center;
				color:#FFFFFF;
				-webkit-transform:scale(0.8);
				font-size:12px;
				cursor:pointer;
			}
			#gnodeb_ctn .alarmCritical{
				background:#E88282;
			}
			#gnodeb_ctn .alarmMajor{
				background:#DCAA5E;
			}
			#gnodeb_ctn .alarmMinor{
				background:#CCCC66;
			}
			#gnodeb_ctn .alarmWarning{
				background:#9AF0FE;
			}
			.el-select-dropdown.is-multiple .el-select-dropdown__item.selected::after{
				right:2px;
			}
			#gnodeb_ctn .el-ctable-toolbar{
				padding: 0px!important;
			}
			.cellNameClass{
	            display: inline-block;
	            height: 18px;
	            width: 18px;
	            text-align: center;
	            line-height: 18px;
	            border: 1px solid #DCDFE6;
	            border-radius: 2px;
	        }
	        .cellNameClass .el-icon::before{
	            font-size: 14px;
	            color: #F2B354;
	            margin-right: 5px;
	        }
	        .cellNameClass:hover{
	            border: 1px solid #4D84FF;
	        }
			.gray-color {
				font-size: 12px;
				color: #999;
			}
			#gnodeb_ctn .col-group ,.gnbMonitorPopoverClass .col-group,.gnbMonitorSyncDialogCls .col-group{
				margin: 5px 10px;
				display: flex;
				flex-wrap: wrap;
			}
			#gnodeb_ctn .col-group .el-checkbox ,.gnbMonitorPopoverClass .col-group .el-checkbox,.gnbMonitorSyncDialogCls .col-group .el-checkbox{
				min-width: 140px;
			}
			#gnodeb_ctn .col-group .el-checkbox__label ,.gnbMonitorPopoverClass .col-group .el-checkbox__label,.gnbMonitorSyncDialogCls .col-group .el-checkbox__label{
				font-size: 12px;
			}
			#gnodeb_ctn .el-ctable-toolbar {
				position: relative;
			}
			#gnodeb_ctn .col-group-cls input ,.gnbMonitorPopoverClass .col-group-cls input,.gnbMonitorSyncDialogCls .col-group-cls input{
				margin-top:-2px;
				margin-bottom:1px;
				vertical-align:middle;
				margin-right:20px;
			}
			#gnodeb_ctn .select-all-cls ,.gnbMonitorPopoverClass .select-all-cls,.gnbMonitorSyncDialogCls .select-all-cls {
				padding: 10px 0 0 9px;
				display: flex;
				align-items: center;
			}
			#gnodeb_ctn .select-all-cls > i ,.gnbMonitorPopoverClass .select-all-cls > i ,.gnbMonitorSyncDialogCls .select-all-cls > i {
				margin-right: 5px;
			}
			#gnodeb_ctn .select-all-cls > span ,.gnbMonitorPopoverClass .select-all-cls > span, .gnbMonitorSyncDialogCls .select-all-cls > span{
				font-size: 14px;
				font-weight: bold;
				margin-left: 10px;
			}
			#gnodeb_ctn .col-group-cls .el-icon-close1::before ,.gnbMonitorPopoverClass .col-group-cls .el-icon-close1::before ,.gnbMonitorSyncDialogCls .col-group-cls .el-icon-close1::before{
				color: #333;
			}
			#gnodeb_ctn .list-group .el-checkbox__label {
				padding-left: 0px;
			}

			#gnodeb_ctn .list-group-item:hover .sort-item-op {
				visibility: visible;
			}
			#gnodeb_ctn .list-group > span {
				display: flex;
				flex-direction: column;
				padding-left: 0px; 
				height: 450px;
				overflow-x: hidden;
			}
			#gnodeb_ctn .list-group-item {
				display: inline-block;
				position: relative;
				padding: 0px 10px;
				margin: 3px 5px;
				width: 260px;
				border: 0px dashed #ddd;
				cursor: move;
			}
			#gnodeb_ctn .gnbCollectMsgBoxCls {
				display: flex;
				top: 4px;
				right: 60px;
				position: absolute;
				border: 1px solid #DFE2EE;
				border-radius: 5px;
			}
			#gnodeb_ctn .gnbCollectMsgBoxCls .foldBtnCls{
				height: 26px;
				width: 26px;
				display: flex;
				flex-direction: column;
				justify-content: center;
				align-items: center;
				background-color: #fff;
				cursor: pointer;
				transform: rotate(90deg);
			}
			#gnodeb_ctn .gnbCollectMsgBoxCls .collect-bt {
				color:#4D84FF;
				margin-left: 15px;
				text-decoration: underline;
				cursor: pointer;
			}
            .moreStatusBoxCls {
                position: relative;
                display: flex;
                max-width: 825px;
                flex-wrap: wrap;
                max-height: 500px;
                overflow: auto;
                padding:0 15px 15px 15px;
            }
            .moreStatusItemCls {
                display: flex;
                padding: 10px;
                border: 1px solid #E9E9E9;
                margin:5px;
                border-radius:2px;
            }
            .moreStatusItemCls .el-icon-status-MME {
                margin-top:20px;
            }
            .moreStatusItemInfoCls {
                display: flex;
                flex-direction: column;
                margin-left:5px;
            }
            .moreStatusPopoverClass .el-popover__title{
                height:35px;
                line-height:35px;
                margin-bottom:0;
                padding:0 20px 0;
            }
			#gnodeb_ctn .remarkHeaderCls ,#gnodeb_ctn .remarkHeaderCls div{
				padding-left: unset;
				line-height: unset;
			}
			#gnodeb_ctn .remarkHeaderCls .el-input__suffix{
				line-height: 30px;
			}
			#gnodeb_ctn .remarkHeaderCls .el-input--suffix .el-input__inner{
				padding-right: 50px;
				box-sizing: border-box;
			}
			#gnodeb_ctn .remarkHeaderCls .el-icon::before{
				font-size: 16px;
				color: #7A7992;
			}
		</style>
	</head>
	<body>
		<div id="gnodeb_ctn" :class="ctnCls">
			<div class="fit commonWarp" style="width: 100%;position: relative;">
				<!-- 表格组件 -->
				<el-ctable ref="monitor"
					id="gnb_monitor_tb"
					:time="6"
					:url="tbURL"
					:limit="limitBatch"
					row-key="small_cell_code"
					@selection-change="selectionChange"
					:query-params="queryParams">
					<!-- 列表toolbar -->
					<template slot="toolbar">
						<div class="toolbarHeadBtnBoxCls">
							<!-- 导出 -->
							<el-popover trigger="click" placement="bottom-end" :append-to-body="false" popper-class="gnbMonitorPopoverClass">
								<div slot="reference" class="newIconBoxCls-bt" style="right:20px;top:5px;" tip="<%=rb.getString("DaoChu")%>">
									<span class="el-icon-operation-export el-icon"></span>
								</div>	
								<div style="border-radius: 3px;background: #fff;box-shadow: 5px 10px 23px 0px rgba(0,0,0,.2);width: 700px;">
									<div class="select-all-cls" style="border-bottom: 1px solid #e9e9e9;padding-bottom: 10px;">
										<span style="margin: 0;font-weight: bold;"><%=rb.getString("DaoChu")%></span>
										<span style="font-size: 12px;font-weight: normal;">Select all parameters that should be included</span>
									</div>
									<div class="flex-ctn col-group-cls" style="height:400px;flex-direction: row;border-bottom: 1px solid #e9e9e9;">
										<div style="padding: 0px 0 0 10px;height:100%;flex:3;overflow:auto;">
											<div class="select-all-cls" style="padding-left: 10px;">
												<el-checkbox :indeterminate="!exportcolGroupAll" v-model="exportcolGroupAll" @change="exportcolGroupAllChange"></el-checkbox> 
												<span><%=rb.getString("QuanXuan")%></span>
											</div>
											<div>
												<div class="select-all-cls">
													<el-checkbox 
														:indeterminate="exportColForm.device.length<colGroups.device.length && exportColForm.device.length>0" 
														v-model="exportdeviceGroupAll" 
														@change="exportdeviceGroupAllChange">
													</el-checkbox> 
													<span><%=rb.getString("SheBeiXinXi")%></span>
												</div>
												<el-checkbox-group class="col-group" v-model="exportColForm.device">
													<el-checkbox v-for="item in colGroups.device" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
												</el-checkbox-group>
											</div>
											<div>
												<div class="select-all-cls">
													<el-checkbox 
														:indeterminate="exportColForm.cell.length<colGroups.cell.length && exportColForm.cell.length>0" 
														v-model="exportcellGroupAll" 
														@change="exportcellGroupAllChange">
													</el-checkbox> 
													<span><%=rb.getString("XiaoQuXinXi")%></span>
												</div>
												<el-checkbox-group class="col-group" v-model="exportColForm.cell">
													<el-checkbox v-for="item in colGroups.cell" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
												</el-checkbox-group>
											</div>
											<div>
												<div class="select-all-cls">
													<el-checkbox 
														:indeterminate="exportColForm.status.length<colGroups.status.length && exportColForm.status.length>0" 
														v-model="exportstatusGroupAll" 
														@change="exportstatusGroupAllChange">
													</el-checkbox> 
													<span><%=rb.getString("ZhuangTai")%></span>
												</div>
												<el-checkbox-group class="col-group" v-model="exportColForm.status">
													<el-checkbox v-for="item in colGroups.status" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
												</el-checkbox-group>
											</div>
											<div>
												<div class="select-all-cls">
													<el-checkbox 
														:indeterminate="exportColForm.network.length<colGroups.network.length && exportColForm.network.length>0" 
														v-model="exportnetworkGroupAll" 
														@change="exportnetworkGroupAllChange">
													</el-checkbox> 
													<span><%=rb.getString("WangLuoSheZhi")%></span>
												</div>
												<el-checkbox-group class="col-group" v-model="exportColForm.network">
													<el-checkbox v-for="item in colGroups.network" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
												</el-checkbox-group>
											</div>
                                            <div>
												<div class="select-all-cls">
													<el-checkbox 
														:indeterminate="exportColForm.location.length<colGroups.location.length && exportColForm.location.length>0" 
														v-model="exportlocationGroupAll" 
														@change="exportlocationGroupAllChange">
													</el-checkbox> 
													<span><%=rb.getString("WeiZhi")%></span>
												</div>
												<el-checkbox-group class="col-group" v-model="exportColForm.location">
													<el-checkbox v-for="item in colGroups.location" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
												</el-checkbox-group>
											</div>
                                            <hr style="border: none;border-bottom: 1px solid #e4e7ec;margin: 10px 0;">
                                            <div style="margin-left: 20px;">
                                                <div style="font-size: 14px;color:#606266;font-weight:bold;"><%=rb.getString("LicenseXinXi")%></div>
                                                <el-checkbox v-model="isExportDeviceLicenseInfo" style="margin: 10px 20px;"><%=rb.getString("DaoChuLicenseXinXi")%></el-checkbox> 
                                            </div>
											
											<hr style="border: none;border-bottom: 1px solid #e4e7ec;margin: 10px 0;">
											<div style="margin-left: 20px;">
												<div style="font-size: 14px;color:#606266;font-weight:bold;"><%=rb.getString("DaoChuGeShi")%></div>
												<el-radio-group v-model="exportFormatType">
													<el-radio style="margin: 10px 20px;" label="csv"><%=rb.getString("CSVGeShi")%></el-radio> 
													<el-radio style="margin: 10px 20px;" label="xlsx"><%=rb.getString("XLSXGeShi")%></el-radio> 
												</el-radio-group>
											</div>
										</div>
                                    </div>
									<div class="windowButtonGroup" style="float:none !important;padding:15px 0 15px 20px;position:relative;z-index:321;">			
										<a class="linkbutton linkbutton_trend" @click="exportForm"><span><%=rb.getString("QueDing")%></span></a>
										<a class="linkbutton linkbutton_nowanna" @click="document.body.click()"><span><%=rb.getString("QuXiao")%></span></a>
									</div>
								</div>
							</el-popover>
							<!-- progress -->
							<div v-if="!progressHide" class="gnbCollectMsgBoxCls" :style="{right: msgExtend? (collectTaskTime==''?'550px':'650px'):'100px', display: 'flex', 'align-items': 'baseline', padding: '5px 0 5px 15px'}">
								<%=rb.getString("DaoChuJinDu")%>: <el-progress :percentage="percentage" style="width:120px;margin-left: 5px;" color="#f56c6c"></el-progress>
							</div>
							<!-- 收集TR069报文 -->
							<div v-if="isGnbCollectExisted && writableMap['CODE_GNB_TR069_MSG_EXCHANGE'] == true" class="gnbCollectMsgBoxCls">
								<div v-if="msgExtend" style="margin-right: 20px;line-height:26px;">
									<span style="padding: 0px 10px;">{{collectActiveSn}}</span>
									<span style="padding: 0px 5px;" v-if="collectTaskTime==''">
										<i class="el-icon el-icon-status-yes" style="font-size: 12px;"></i>
										<%=rb.getString("ChengGong")%>
									</span>
									<span v-if="collectTaskTime!=''" style="display: inline-block;padding: 2px 30px;background: #4d84ff;border-radius: 2px;margin: 0px 5px 2px 5px;"></span>
									<span v-if="collectTaskTime!=''" style="border: 1px solid #e3e3e3;border-radius: 3px;padding: 2px 4px;">
										<span style="cursor: pointer;" @click="stopCollect">
											<i style="padding: 4px;background: red;height: 0px;display: inline-block;border-radius: 3px;"></i>
											<%=rb.getString("TingZhi")%>
										</span>
										<span style="margin-left: 5px;">
											{{collectTaskTime}}
										</span>
									</span>
									<span style="margin-left: 20px;">
										<a class="collect-bt" @click="viewMsg"><%=rb.getString("ChaKan")%></a>
										<a class="collect-bt" @click="downloadMsg"><%=rb.getString("XiaZai")%></a>
										<a class="collect-bt" @click="clearMsg"><%=rb.getString("QingChu")%></a>
									</span>
								</div>
								<div @click="msgExtend = !msgExtend" class="foldBtnCls">
									<span v-if="msgExtend" class="el-icon el-icon-common-query-up"></span>
									<span v-if="!msgExtend" class="el-icon el-icon-common-query-down"></span>
								</div>
							</div>
							<div v-show="isMonitorWritable" class="selectBlukBoxCls">
								<div class="selectMain">
									<div class="bulkSelectBtnBoxCls"  @click="openBulkSelectTable">
										<span class="el-icon-selected el-icon"></span>
										<span class="bulkSelectNumBoxCls">( {{selectedRows.length}} )</span>
									</div>
									<div class="selectTableBoxCls" style="position: absolute;top: 38px;left: 0px;" v-show="bulkSelectShow">
										<div class="selectBoxTitle">
											<span><%=rb.getString("YiXuan")%></span>
											<span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="closeBulkSelectTable"></span>
										</div>
										<div class="selectBoxMain">
											<div class="tableInfoCls">
												<div class="tableInfoHeader">
													<div><%=rb.getString("Title_SheBeiBianMa")%></div>
													<div @click="clearBulkSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span>Clear</div>
												</div>
												<el-ctable 
													id="bulkSelectTable" 
													ref="bulkSelectTable" 
													:data="selectedRows" 
													:showHeader="false"
													:rownumber="false"
													:front-pagination="true"
													height="300px" pagination="true" >
													<el-table-column prop="id" v-if="false"></el-table-column>
													<el-table-column width="588">
														<template slot-scope="scope" >
															<div class="tableItemCls">
																<span>{{scope.row.serial_number}}</span>
																<span @click="delBulkSelected(scope.row)" class="el-icon el-icon-circle-close item_show"></span>
															</div>
														</template>
													</el-table-column>
												</el-ctable>
											</div>
										</div>
									</div>
								</div>
							</div>
                            <div v-if="writableMap['CODE_GNB_SYNCHRONIZE'] == true" :class="selectedRows.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="batchSyncList">
                                <span class="el-icon el-icon-operation-synchronize"></span>
                                <span><%=rb.getString("TongBu")%></span>
                            </div>
                            <div v-if="isMonitorWritable && writableMap['CODE_GNB_REBOOT'] == true" :class="selectedRows.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="batchRebootList">
                                <span class="el-icon el-icon-operation-reboot"></span>
                                <span><%=rb.getString("ChongQi")%></span>
                            </div>
							<div v-if="isMonitorWritable && writableMap['CODE_GNB_DEVICE_REGISTER'] == true" :class="selectedRows.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="recycleCells">
								<span class="el-icon el-a-icon-Recyclebin"></span>
								<span><%=rb.getString("HuiShouZhan")%></span>
							</div>
						</div>
						<span class="el-icon-operation-settings el-icon" style="position:absolute;left:5px;z-index:99;box-shadow:none;bottom:-27px;" @click="columnSetting"></span>
						<div v-show="sortGroupShow" style="border-radius: 3px;position: absolute; top: 85px; left: 0px; z-index: 1000;background: #fff;box-shadow: 5px 10px 23px 0px rgba(0,0,0,.2);width: 900px;">
							<div class="flex-ctn col-group-cls" id="gnbSortAndShowColumnBoxCls" style="height:500px;flex-direction: row;border-bottom: 1px solid #e9e9e9;">
								<div style="padding: 0px 0 0 10px;height:100%;flex:3;overflow:auto;">
									<div class="select-all-cls">
										<span style="margin: 0;font-weight: bold;"><%=rb.getString("XuanZeLie")%></span>
									</div>
									<div class="select-all-cls" style="padding-left: 10px;">
										<el-checkbox :indeterminate="!colGroupAll" v-model="colGroupAll" @change="colGroupAllChange"></el-checkbox> 
										<span><%=rb.getString("QuanXuan")%></span>
									</div>
									<div>
										<div class="select-all-cls">
											<i :class="{'el-icon':true,'el-icon-close1':colExpanded.device,'el-icon-open':!colExpanded.device}" @click="colExpanded.device = !colExpanded.device"></i>
											<el-checkbox :indeterminate="colForm.device.length<colGroups.device.length && colForm.device.length>0" v-model="deviceGroupAll" @change="deviceGroupAllChange"></el-checkbox> 
											<span><%=rb.getString("SheBeiXinXi")%></span>
										</div>
										<el-checkbox-group v-show="colExpanded.device" class="col-group" v-model="colForm.device">
											<el-checkbox v-for="item in colGroups.device" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
										</el-checkbox-group>
									</div>
									<div>
										<div class="select-all-cls">
											<i :class="{'el-icon':true,'el-icon-close1':colExpanded.cell,'el-icon-open':!colExpanded.cell}" @click="colExpanded.cell = !colExpanded.cell"></i>
											<el-checkbox :indeterminate="colForm.cell.length<colGroups.cell.length && colForm.cell.length>0" v-model="cellGroupAll" @change="cellGroupAllChange"></el-checkbox> 
											<span><%=rb.getString("XiaoQuXinXi")%></span>
										</div>
										<el-checkbox-group v-show="colExpanded.cell" class="col-group" v-model="colForm.cell">
											<el-checkbox v-for="item in colGroups.cell" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
										</el-checkbox-group>
									</div>
									<div>
										<div class="select-all-cls">
											<i :class="{'el-icon':true,'el-icon-close1':colExpanded.status,'el-icon-open':!colExpanded.status}" @click="colExpanded.status = !colExpanded.status"></i>
											<el-checkbox :indeterminate="colForm.status.length<colGroups.status.length && colForm.status.length>0" v-model="statusGroupAll" @change="statusGroupAllChange"></el-checkbox> 
											<span><%=rb.getString("ZhuangTai")%></span>
										</div>
										<el-checkbox-group v-show="colExpanded.status" class="col-group" v-model="colForm.status">
											<el-checkbox v-for="item in colGroups.status" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
										</el-checkbox-group>
									</div>
									<div>
										<div class="select-all-cls">
											<i :class="{'el-icon':true,'el-icon-close1':colExpanded.network,'el-icon-open':!colExpanded.network}" @click="colExpanded.network = !colExpanded.network"></i>
											<el-checkbox :indeterminate="colForm.network.length<colGroups.network.length && colForm.network.length>0" v-model="networkGroupAll" @change="networkGroupAllChange"></el-checkbox> 
											<span><%=rb.getString("WangLuoSheZhi")%></span>
										</div>
										<el-checkbox-group v-show="colExpanded.network" class="col-group" v-model="colForm.network">
											<el-checkbox v-for="item in colGroups.network" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
										</el-checkbox-group>
									</div>
                                    <div>
                                        <div class="select-all-cls">
                                            <i :class="{'el-icon':true,'el-icon-close1':colExpanded.location,'el-icon-open':!colExpanded.location}" @click="colExpanded.location = !colExpanded.location"></i>
                                            <el-checkbox :indeterminate="colForm.location.length<colGroups.location.length && colForm.location.length>0" v-model="locationGroupAll" @change="locationGroupAllChange"></el-checkbox> 
                                            <span><%=rb.getString("WeiZhi")%></span>
                                        </div>
                                        <el-checkbox-group v-show="colExpanded.location" class="col-group" v-model="colForm.location">
                                            <el-checkbox v-for="item in colGroups.location" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}">{{item.label}}</el-checkbox>
                                        </el-checkbox-group>
                                    </div>
								</div>
								
								<div style="border-left: 1px solid #e9e9e9;height:100%;flex:1;width: 290px;">
									<div class="select-all-cls" style="margin: 10px; padding: 0px; border-bottom: 1px solid #e9e9e9;">
										<span style="padding-bottom: 10px;margin-left: 0px;font-weight: bold;"><%=rb.getString("LiePaiXu")%></span>
									</div>
									<el-checkbox-group v-model="dragCol">
										<draggable
											class="list-group"
											v-model="dragColumns"
											v-bind="dragOptions">
											<transition-group type="transition" :name="!drag? 'flip-list':null">
												<div v-for="(col,idx) in dragColumnsWithLabel" :key="col.code" class="list-group-item" v-if="showCols.includes(col.code)">
													<span class="el-checkbox__label" style="border-bottom: 1px dashed #e9e9e9;min-width: 260px;color: #606266;">
														{{col.label}}
						
														<i v-if="!col.disabled" class="el-icon el-icon-close sort-item-op" style="zoom: 0.6;float: right; margin: 6px 10px 0px 0px;cursor: pointer;" @click="clickColumnLabel(col.label)"></i>
													</span>
													<el-checkbox v-if="false" :label="col.code" :key="col.code" :disabled="col.disabled">{{col.label}} </el-checkbox>
												</div>
											</transition-group>
										</draggable>
									</el-checkbox-group>
								</div>
							</div>
							<div class="windowButtonGroup" style="float:none !important;padding:15px 0 15px 20px;position:relative;z-index:321;">			
								<a class="linkbutton linkbutton_trend" @click="ColumnConfigEn()"><span><%=rb.getString("QueDing")%></span></a>
								<a class="linkbutton linkbutton_nowanna" @click="close"><span><%=rb.getString("QuXiao")%></span></a>
							</div>
						</div>
						<!-- 高级查询 -->
						<div id="tableHeadQuery" class="tableHeadQueryBoxCls" style='margin-left: 10px;'>
							<div class="headQueryBox">
								<div class="queryGroup">
									<el-input v-model="search_text" @keyup.enter.native="query" @focus="queryInputFocus" @blur="queryInputBlur" :placeholder='placeholderText' style="width:260px;"></el-input>
									<i @click='query' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
								</div>
							</div>
							<div v-for="(item,index) in advancedQueryItemList">
								<div v-if="item.type == 'checkbox' && item.isShow" style="margin-right:10px;">
									<el-popfilter
										:label='item.label'
										v-model="item.checkedItemList"
										:list="item.options"
										:visible.sync="item.isShow"
										:closable="true"
										@check-change="advanceQuery(item.type,item.value,item.checkedItemList)"
										@close="checkItemDel(item)">
									</el-popfilter>
								</div>
								<div v-if="item.type == 'select' && item.isShow" style="margin-right:10px;">
									<el-popfilter
										type="single"
										:label='item.label'
										v-model="item.selectVal"
										:list="item.options"
										:visible.sync="item.isShow"
										:closable="true"
										@check-change="advanceQuery(item.type,item.value,item.selectVal)"
										@close="checkItemDel(item)">
									</el-popfilter>
								</div>
								<div v-if="item.type == 'filter'" style="margin-right:10px;">
									<div class="advancedQueryItemBox" style="background: #FFF;">
										<el-popover :ref="'popover-'+item.value" trigger="click" placement="bottom-start"  @show="checkPopoverShow(item)" @hide="checkPopoverHide(item,'')">
											<div class="checkPopoverBoxCls">
												<el-checkbox-group v-model="item.checkedItemList" @change="handleCheckedChange(item)">
													<el-checkbox v-for=" items in item.options" :key="items.value" :label="items.value">{{items.label}}</el-checkbox>
												</el-checkbox-group>
												<div class="buttonGroup">
													<el-button size="mini" type="primary" @click="checkPopoverSubmit(item)"><%=rb.getString("QueDing")%></el-button>
													<el-button size="mini" @click="checkPopoverHide(item,'del')"><%=rb.getString("QuXiao")%></el-button>
												</div>	
											</div>
											<div slot="reference" class="ItemAndIconBoxCls">
												<i class="el-icon el-icon-filterAdd"></i>
												{{item.label}}
											</div>
										</el-popover>
									</div>
								</div>
							</div>
							<div class="advancedQueryItemBox"  style="background: #FFF;" @click="clearFilterClick">
								<%=rb.getString("QingKongShaiXuan")%>
							</div>
						</div>
					</template>
					<!-- 列表columns -->
					<el-table-column v-if="isMonitorWritable" key="selection" type="selection" :reserve-selection="true" fixed></el-table-column>
					<el-table-column prop="op" key="op" label=" " width="80" align="center" fixed>
						<template slot-scope="scope">
							<div class="el-icon el-icon-operation-settings" @click="settingBtnClick(scope.row,'info')" style='margin-right: 10px;'></div>
							<div v-show="isMonitorWritable" class="el-icon el-icon-operation-more-circle" v-clickoutside="hideMenus" @click="operClick(event,scope.row)"></div>
						</template>
					</el-table-column>
					<el-table-column prop="connection_status" key="connection_status" label=" " sortable width="60" fixed>
						<template slot-scope="scope">
							<div v-html="connStatusFormatterSyn(scope.row.connection_status, scope.row)"></div>
						</template>
					</el-table-column>
					<el-table-column prop="alarm_count" key="alarm_count" width="110" label="<%=rb.getString("GaoJingShu")%>" sortable fixed>
						<template slot-scope="scope">
	                        <div style="text-align: center;">
	                            <span v-if="scope.row.alarm_serverity == '31001' "  class="alarmCritical alarmListSty" @click="alarmToInfo(scope.row, event)">{{scope.row.alarm_count}}</span>
	                            <span v-else-if="scope.row.alarm_serverity == '31002' "  class="alarmMajor alarmListSty" @click="alarmToInfo(scope.row, event)">{{scope.row.alarm_count}}</span>
	                            <span v-else-if="scope.row.alarm_serverity == '31003' "  class="alarmMinor alarmListSty" @click="alarmToInfo(scope.row, event)">{{scope.row.alarm_count}}</span>
	                            <span v-else-if="scope.row.alarm_serverity == '31004' "  class="alarmWarning alarmListSty" @click="alarmToInfo(scope.row, event)">{{scope.row.alarm_count}}</span>
	                            <span v-else >0</span>
	                        </div>
	                    </template>
					</el-table-column>
					<el-table-column prop="serial_number" key="serial_number" label="<%=rb.getString("XiaoZhanBianMa")%>" sortable min-width="240" fixed></el-table-column>
					<el-table-column prop="host_name" key="host_name" label="<%=rb.getString("5GZhanDianMingCheng")%>" sortable width="180" fixed>
					    <template slot-scope="scope">
			                        <div v-if="scope.row.device_name_tip === '0'">{{scope.row.host_name}}</div>
			                        <div v-if="scope.row.device_name_tip !== '0'" class="cellNameClass">
			                            <el-popover trigger="click">
			                                <div slot="reference">
			                                    <span class="el-icon el-icon-circle-warning"></span> 
			                                    {{scope.row.host_name}}
			                                </div>
			                                <div style='padding: 30px 20px 20px; position:relative;'>
			                                    <div> <span class="el-icon el-icon-close" @click="closeSyncName" style="top: 10px; position:absolute;"></span>
			                                    <div style='display: flex;'><span style='color: #333; font-size: 14px;'><%=rb.getString("JiZhanCeMingCheng")%> : </span> <span style='font-size: 14px;margin-left: 6px;color: #333;'>{{scope.row.report_host_name}}</span></div>
			                                    <div style='font-size: 12px; color: #333; margin-top: 6px;'><%=rb.getString("ShiFouTongBuMingChengDaoOMC")%><div>
			                                    <div style="margin-top: 10px; text-align: right;">
			                                        <span class="el-button--primary el-button" @click="syncName(scope.row.small_cell_code, scope.row.report_host_name)">
			                                            <%=rb.getString("QueDing")%>
			                                        </span>
			                                        <span class="white el-button" @click="closeSyncName"><%=rb.getString("QuXiao")%></span>
			                                    </div>
			                                </div>
			                            </el-popover>
			                        </div>
			                    </template>
					</el-table-column>
					<el-table-column prop="op_state" key="op_state" label="<%=rb.getString("gNBZhuangTai")%>" sortable width="120">
						<template slot-scope="scope">
							<div v-if="scope.row.op_state && scope.row.op_state.length>1" style="display: flex;">
								<span v-if="judgeActiveStatusFat(scope.row.op_state) == '1'" style='margin-right: 5px;'><%= rb.getString("JiHuo")%></span>
								<span v-if="judgeActiveStatusFat(scope.row.op_state) == '2'" class='offOrOnStatusCls' style='margin-right: 5px;'><%= rb.getString("JiHuo")%></span>
								<span v-if="judgeActiveStatusFat(scope.row.op_state) == '3'" class='offStatusCls' style='margin-right: 5px;'><%= rb.getString("QuJiHuo")%></span>
								<el-popover title="<%= rb.getString("DuoXiaoQuZhuangTai")%>" popper-class="cellActivePopoverClass" trigger="click" width='300'>
									<span style="color:#4d84ff;" slot="reference">
										[ {{judgeActiveNumOrAllNumFat(scope.row.op_state,'active')}}/{{judgeActiveNumOrAllNumFat(scope.row.op_state,'all')}} ]
									</span>
									<div style="padding:10px;border-top:1px solid #E9E9E9;min-height:100px;display:flex;flex-wrap:wrap;">
										<div v-for="(item,index) in parseCellAndRfStatus(scope.row.op_state)" style="height:40px;display:flex;align-items: center;margin-left:10px;">
											Cell {{index+1}}:
											<span v-if=" item == '1'" class="onStatusBoxCls" style='margin-left: 5px;'><%= rb.getString("JiHuo")%></span>
											<span v-if=" item == '0'" class="offStatusBoxCls" style='margin-left: 5px;'><%= rb.getString("QuJiHuo")%></span>
										</div>
									</div>
								</el-popover>
							</div>
							<div v-else v-html="cellStateFormatter(scope.row.op_state, scope.row)"></div>
						</template>
					</el-table-column>
					<el-table-column prop="PHYCELLID" key="PHYCELLID" label="<%=rb.getString("PCI2")%>" sortable width="120"></el-table-column>
					<el-table-column prop="cell_ip" key="cell_ip" label="<%=rb.getString("IPDiZhi")%>" sortable width="120"></el-table-column>
                    <el-table-column prop="mac_address" key="mac_address" sortable label="MAC"  width="130"></el-table-column>
					<el-table-column v-if="showColums.includes('product_name')" prop="product_name" key="product_name" label="<%=rb.getString("ChanPinMingCheng")%>" sortable width="120"></el-table-column>
					<el-table-column prop="module_type" key="module_type" label="<%=rb.getString("SheBeiXingHaoMing")%>" sortable width="120"></el-table-column>
					<el-table-column prop="product" key="product" label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" sortable width="120"></el-table-column>
					<el-table-column prop="software_version" key="software_version" label="<%=rb.getString("RuanJianBanBen")%>" sortable width="150">
						<template slot-scope="scope">
							<div>{{scope.row.software_version}}</div>
						</template>
					</el-table-column>
                    <el-table-column v-if="showColums.includes('amf_status')" key="amf_status" label="AMF Status" prop="amf_status" width="140">
                        <template slot-scope="scope">
                            <div v-if="scope.row.product == 'BaiBNQ'">
                                <div v-if="['','NULL','null',null,undefined].includes(scope.row.amf_status)">--</div>
                                <div v-if="!['','NULL','null',null,undefined].includes(scope.row.amf_status)" style="display: flex;align-items: center;">
                                    <span v-html="amfStatusFmt(scope.row.amf_status)" ></span>
                                    <el-popover title="All AMF Status" popper-class="moreStatusPopoverClass">
                                        <span style="color:#4d84ff;cursor:pointer;" slot="reference">
                                            [ {{parseParamsOnNum(scope.row.amf_status)}}/{{parseParams(scope.row.amf_status).length}} ]
                                        </span>
                                        <div class="moreStatusBoxCls">
                                            <i class="el-icon el-icon-close" style="position: absolute; top: -30px; right: 0px;" onclick="document.body.click()"></i>
                                            <div class="moreStatusItemCls" v-for="item in parseParams(scope.row.amf_status)">
                                                <span v-if="item.Status == '1'" class="el-icon el-icon-status-MME greenIcon"></span>
                                                <span v-if="item.Status == '0'" class="el-icon el-icon-status-MME redIcon"></span> 
                                                <div class="moreStatusItemInfoCls">
                                                    <span>AMF IP : {{item.AmfIP1}}</span>
                                                    <span v-if="item.Status==='0'">AMF Status : <%= rb.getString("MMEWeiLianJie")%></span>
                                                    <span v-if="item.Status=='1'">AMF Status : <%= rb.getString("MMEYiLianJie")%></span>
                                                    <span v-if="item.Status=='2'">AMF Status : --</span>
                                                    <span v-if="item.Status==='' || item.Status===null">AMF Status : </span>
                                                    <span><%=rb.getString("PLMN")%> : {{item.PLMNID}}</span>
                                                </div>
                                            </div>
                                        </div>                                  
                                    </el-popover>
                                </div>                                   
                            </div>
                            <div v-else>--</div>
                        </template>
                    </el-table-column>
					<el-table-column prop="halob_flag" key="halob_flag" label="<%=rb.getString("HaloBKaiGuan")%>" sortable width="120">
						<template slot-scope="scope">
							 <div v-if="scope.row.halob_flag == '1'" style="display: flex;align-items: center;">
								HaloB<span class='el-icon el-icon-status-enable' style='margin-left:3px;font-size: 20px;'></span>
							</div>
							<div v-else-if="scope.row.halob_flag == '0'" style="display: flex;align-items: center;">
								HaloB<span class='el-icon el-icon-status-disable' style='margin-left:3px;font-size: 20px;'></span>
							</div>
							<div v-else>--</div>
						</template>
					</el-table-column>
					<el-table-column prop="group_name" key="group_name" label="<%=rb.getString("SheBeiZu")%>" sortable min-width="120">
						<template slot-scope="scope">
							<div>{{scope.row.group_name}}</div>
						</template>
					</el-table-column>
					<el-table-column label="<%=rb.getString("UEShu")%>" prop="ue_count" key="ue_count" sortable width="80">
						<template slot-scope="scope">
							<div v-html="ueCountsFormatter(scope.row, scope.row.ue_count, scope.$index)"></div>
						</template>
					</el-table-column>
					<el-table-column v-if="siteEnable && showColums.includes('sub_station_name')" key="sub_station_name"  label='<%=rb.getString("ZhanZhiMingCheng")%>' prop="sub_station_name" width="120"></el-table-column>
					<el-table-column v-if="showColums.includes('rf_status')" key="rf_status" label="<%=rb.getString("ShePinKaiGuanZhuangTai")%>" prop="rf_status" width="150" :show-overflow-tooltip="false">
						<template slot-scope="scope">
							<div style="display: flex;align-items: center;">
								<div v-if="['','NULL','null',null,undefined].includes(scope.row.rf_status)"></div>
								<div v-if="scope.row.rf_status == '--'">--</div>
								<div v-if="!['','NULL','null',null,undefined].includes(scope.row.rf_status) && scope.row.rf_status != '--'">
									<div v-if="scope.row.rf_status == 'on' && parseCellAndRfStatus(scope.row.rf_status).length == 1" class='iconFlexCls'>
										<span style='margin-right: 5px;'><%=rb.getString("Kai")%></span>
									</div>
									<div v-if="scope.row.rf_status == 'off' && parseCellAndRfStatus(scope.row.rf_status).length == 1" class='iconFlexCls'>
										<span class='offStatusCls' style='margin-right: 5px;'><%=rb.getString("Guan")%></span>
									</div>
									<div v-if="parseCellAndRfStatus(scope.row.rf_status).length > 1 "  style="display: flex;">
										<span v-if="judgeActiveStatusFat(scope.row.rf_status) == '1'" style='margin-right: 5px;'><%=rb.getString("Kai")%></span>
										<span v-if="judgeActiveStatusFat(scope.row.rf_status) == '2'" class='offOrOnStatusCls' style='margin-right: 5px;'><%=rb.getString("Kai")%></span>
										<span v-if="judgeActiveStatusFat(scope.row.rf_status) == '3'" class='offStatusCls' style='margin-right: 5px;'><%=rb.getString("Guan")%></span>
										<el-popover title="<%= rb.getString("DuoXiaoQuZhuangTai")%>" popper-class="cellActivePopoverClass" trigger="click" width='300'>
											<span style="color:#4d84ff;" slot="reference" v-if="parseCellAndRfStatus(scope.row.rf_status).length>1">
												[ {{judgeActiveNumOrAllNumFat(scope.row.rf_status,'active')}}/{{judgeActiveNumOrAllNumFat(scope.row.rf_status,'all')}} ]
											</span>
											<div style="padding:10px;border-top:1px solid #E9E9E9;min-height:100px;display:flex;flex-wrap:wrap;">
												<div v-for="(item,index) in parseCellAndRfStatus(scope.row.rf_status)" style="height:40px;display:flex;align-items: center;margin-left:10px;">
													Cell {{index+1}}:
													<span v-if=" item == 'on'" class="onStatusBoxCls" style='margin-left: 5px;'><%=rb.getString("Kai")%></span>
													<span v-if=" item == 'off'" class="offStatusBoxCls" style='margin-left: 5px;'><%=rb.getString("Guan")%></span>
												</div>
											</div>
										</el-popover>
									</div>
								</div>
							</div>
						</template>
					</el-table-column>
					<el-table-column v-if="showColums.includes('gNBId')" key="gNBId" label="<%=rb.getString("GnodebId")%>" prop="gNBId" sortable width="80"></el-table-column>
					<el-table-column v-if="showColums.includes('firmware_version')" key="firmware_version" label="Hardware Version" prop="firmware_version" sortable width="150"></el-table-column>
					<el-table-column v-if="showColums.includes('up_time')" key="up_time" label="<%=rb.getString("YunXingShiJian")%>" prop="up_time" sortable width="120"></el-table-column>
					<el-table-column v-if="showColums.includes('first_online_time')" key="first_online_time" label="<%=rb.getString("DiYiCiLianJieShiJian")%>" prop="first_online_time" sortable width="150"></el-table-column>
					<el-table-column v-if="showColums.includes('remark')" key="remark" prop="remark" width="185">
						<template slot="header" slot-scope="scope">
							<div class="remarkHeaderCls">
								<span v-if="!editingRemarkLabel"  style="display: flex; align-items:center;">
									<span :title="currentRemarkLabel" style="max-width: 140px; display: inline-block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; vertical-align: middle;">
										{{currentRemarkLabel}}
									</span>
									<i class="el-icon el-icon-operation-edit" @click="startEditRemarkLabel" style="margin-left: 5px;cursor: pointer;"></i>
								</span>
								<span v-else>
									<el-input v-model="remarkLabelInput" size="mini" style="width: 150px;" maxlength="30" @keyup.enter.native="saveRemarkLabel">
										<template slot="suffix">
											<i @click="saveRemarkLabel" class="el-icon el-icon-operation-defaultBeta" style="margin-right: 5px;cursor: pointer;"></i>
											<i @click="cancelEditRemarkLabel" class="el-icon el-icon-deactivate" style="cursor: pointer;"></i>
										</template>
									</el-input>
								</span>
							</div>
						</template>
					</el-table-column>
					<el-table-column v-if="showColums.includes('LASTINFORMTIME')" key="LASTINFORMTIME" label="<%=rb.getString("ShangCiLianJieShiJian")%>" prop="LASTINFORMTIME" sortable width="150"></el-table-column>
					<el-table-column v-if="showColums.includes('online_time')" key="online_time" sortable label="<%=rb.getString("JieRuShiJian")%>" prop="online_time" width="140"></el-table-column>
					<el-table-column v-if="showColums.includes('offline_time')" key="offline_time" sortable label="<%=rb.getString("DuanKaiShiJian")%>" prop="offline_time" width="140"></el-table-column>
					<el-table-column v-if="showColums.includes('network_model')" key="network_model" sortable label="<%=rb.getString("JiZhanZhiShi")%>" prop="network_model" width="120"></el-table-column>
                    <el-table-column v-if="showColums.includes('nr_cell_id')" key="nr_cell_id" label="<%=rb.getString("NRXiaoQuID")%>" prop="nr_cell_id" sortable width="100"></el-table-column>
					<el-table-column v-if="showColums.includes('tac')" key="tac" label="<%=rb.getString("TAC")%>" prop="tac" sortable width="80"></el-table-column>
					<el-table-column v-if="showColums.includes('Band')" key="Band" label="<%=rb.getString("PinDuan")%>" prop="Band" sortable width="140"></el-table-column>
					<el-table-column v-if="showColums.includes('EARFCNULINUSE')" key="EARFCNULINUSE" label="<%=rb.getString("NRPinDianShangXian")%>" prop="EARFCNULINUSE" sortable width="120"></el-table-column>
					<el-table-column v-if="showColums.includes('EARFCNDLINUSE')" key="EARFCNDLINUSE" label="<%=rb.getString("NRPinDianXiaXian")%>" prop="EARFCNDLINUSE" sortable width="120"></el-table-column>
					<el-table-column v-if="showColums.includes('tx_power')" key="tx_power" label="<%=rb.getString("CPETxPower")%>" prop="tx_power" sortable width="80"></el-table-column>
					<el-table-column v-if="showColums.includes('adminState')" key="adminState" label="<%=rb.getString("AdminZhuangTai")%>" prop="adminState" sortable width="120">
						<template slot-scope="scope">
							<span v-if="scope.row.adminState == '1'">Locked</span>
							<span v-if="scope.row.adminState == '2'">Unlocked</span>
							<span v-if="scope.row.adminState == '3'">ShuttingDown</span>
						</template>
					</el-table-column>
					<el-table-column v-if="showColums.includes('synStatus')" key="synStatus" label="<%=rb.getString("TongBuZhuangTai")%>" prop="synStatus" sortable width="120"></el-table-column>
					<el-table-column v-if="showColums.includes('multiPlmnEnable')" key="multiPlmnEnable" label="<%=rb.getString("MultiPLMNZhuangTai")%>" prop="multiPlmnEnable" sortable width="150">
						<template slot-scope="scope">
							<span v-if="scope.row.multiPlmnEnable == '0'"><%= rb.getString("JinYong")%></span>
							<span v-if="scope.row.multiPlmnEnable == '1'"><%= rb.getString("QiYong")%></span>
						</template>
					</el-table-column>
                    <el-table-column v-if="showColums.includes('IPSEC_ADDR')" key="IPSEC_ADDR" label="<%=rb.getString("IPSECDiZhi")%>" prop="IPSEC_ADDR" width="250"></el-table-column>
                    <el-table-column v-if="showColums.includes('gps_longitude')" key="gps_longitude" label="<%=rb.getString("GPSJingDu")%>" prop="gps_longitude" width="100">
                        <template slot-scope="scope">
                            <div v-if="scope.row.gps_modify_flag != '1'">{{scope.row.gps_longitude}}</div>
                            <div v-else>
                                <div v-if="scope.row.gps_longitude === undefined" :key="scope.$index">
                                    {{scope.row.modify_longitude == undefined? scope.row.gps_longitude : scope.row.modify_longitude}}
                                </div>
                                <div v-if="scope.row.gps_longitude !== undefined" class="cellNameClass" :key="scope.$index">
                                    <el-popover trigger="click">
                                        <div slot="reference">
                                            <span class="el-icon el-icon-circle-warning"></span> 
                                            {{scope.row.modify_longitude}}
                                        </div>
                                        <div class="gpsSyncPopoverContent">
                                            <span @click="closeSyncName" class="el-icon el-icon-close gpsSyncPopoverClose"></span>
                                            <div style="margin-bottom: 10px;">
                                                <%=rb.getString("JingDu")%>: {{scope.row.gps_longitude}}&nbsp;&nbsp;
                                                <%=rb.getString("WeiDu")%>: {{scope.row.gps_latitude}}&nbsp;&nbsp;
                                                <%=rb.getString("GaoDu")%>: {{scope.row.gps_height}}
                                            </div>
                                            <div style="margin-bottom: 10px;"><%=rb.getString("TongBuGPSTiShi")%></div>
                                            <div style="text-align: right;margin: 0px;">
                                                <el-button type="primary" size="mini" @click="synchronizeGPS(scope.row.small_cell_code)">
                                                    <%=rb.getString("QueDing")%>
                                                </el-button>
                                                <el-button size="mini" @click="closeSyncName"><%=rb.getString("QuXiao")%></el-button>
                                            </div>
                                        </div>
                                    </el-popover>
                                </div>
                            </div>
                        </template>
                    </el-table-column>
                    <el-table-column v-if="showColums.includes('gps_latitude')" key="gps_latitude" label="<%=rb.getString("GPSWeiDu")%>" prop="gps_latitude" width="90">
                        <template slot-scope="scope">
                            <div v-if="scope.row.gps_modify_flag != '1'">{{scope.row.gps_latitude}}</div>
                            <div v-else>
                                <div v-if="scope.row.gps_latitude === undefined" :key="scope.$index">
                                    {{scope.row.modify_latitude == undefined? scope.row.gps_latitude : scope.row.modify_latitude}}
                                </div>
                                
                                <div v-if="scope.row.gps_latitude !== undefined" class="cellNameClass" :key="scope.$index">
                                    <el-popover trigger="click">
                                        <div slot="reference">
                                            <span class="el-icon el-icon-circle-warning"></span> 
                                            {{scope.row.modify_latitude}}
                                        </div>
                                        <div class="gpsSyncPopoverContent">
                                            <span @click="closeSyncName" class="el-icon el-icon-close gpsSyncPopoverClose"></span>
                                            <div style="margin-bottom: 10px;">
                                                <%=rb.getString("JingDu")%>: {{scope.row.gps_longitude}}&nbsp;&nbsp;
                                                <%=rb.getString("WeiDu")%>: {{scope.row.gps_latitude}}&nbsp;&nbsp;
                                                <%=rb.getString("GaoDu")%>: {{scope.row.gps_height}}
                                            </div>
                                            <div style="margin-bottom: 20px;"><%=rb.getString("TongBuGPSTiShi")%></div>
                                            <div style="text-align: right;margin: 0px;">
                                                <el-button type="primary" size="mini" @click="synchronizeGPS(scope.row.small_cell_code)">
                                                    <%=rb.getString("QueDing")%>
                                                </el-button>
                                                <el-button size="mini" @click="closeSyncName"><%=rb.getString("QuXiao")%></el-button>
                                            </div>
                                        </div>
                                    </el-popover>
                                </div>
                            </div>
                        </template>
                    </el-table-column>
                    <el-table-column v-if="showColums.includes('gps_height')" key="gps_height" label="<%=rb.getString("GPSGaoDu")%>" prop="gps_height" width="70">
                        <template slot-scope="scope">
                            <div v-if="scope.row.gps_modify_flag != '1'">{{scope.row.gps_height}}</div>
    
                            <div v-else>
                                <div v-if="scope.row.gps_height === undefined" :key="scope.$index">
                                    {{scope.row.modify_height == undefined? scope.row.gps_height : scope.row.modify_height}}
                                </div>
                                
                                <div v-if="scope.row.gps_height !== undefined" class="cellNameClass" :key="scope.$index">
                                    <el-popover trigger="click">
                                        <div slot="reference">
                                            <span class="el-icon el-icon-circle-warning"></span> 
                                            {{scope.row.modify_height}}
                                        </div>
                                        <div class="gpsSyncPopoverContent">
                                            <span @click="closeSyncName" class="el-icon el-icon-close gpsSyncPopoverClose"></span>
                                            <div style="margin-bottom: 10px;">
                                                <%=rb.getString("JingDu")%>: {{scope.row.gps_longitude}}&nbsp;&nbsp;
                                                <%=rb.getString("WeiDu")%>: {{scope.row.gps_latitude}}&nbsp;&nbsp;
                                                <%=rb.getString("GaoDu")%>: {{scope.row.gps_height}}
                                            </div>
                                            <div style="margin-bottom: 20px;"><%=rb.getString("TongBuGPSTiShi")%></div>
                                            <div style="text-align: right;margin: 0px;">
                                                <el-button type="primary" size="mini" @click="synchronizeGPS(scope.row.small_cell_code)">
                                                    <%=rb.getString("QueDing")%>
                                                </el-button>
                                                <el-button size="mini" @click="closeSyncName"><%=rb.getString("QuXiao")%></el-button>
                                            </div>
                                        </div>
                                    </el-popover>
                                </div>
                            </div>
                        </template>
                    </el-table-column>
                    <!-- <el-table-column v-if="showColums.includes('authCode')" key="authCode" label="<%=rb.getString("JianQuanMa")%>" prop="authCode" width="120"></el-table-column> -->
                </el-ctable>
			</div>
			<!-- 菜单 -->
			<el-cmenu ref="menu"
				@click="menuClick"
				:data="menus"></el-cmenu>

			<el-slide ref="slide" :title="slideTitle" :url="slideURL" :header='false' :footer="false" :position="slidePosition" :height="sliderHeight" :width="sliderWidth"  
				class="no-padding commonSlideClass commonWarp" @cancel="hideSlide"></el-slide>
				
			<!-- settingSlide -->
			<el-slide  ref="settingSlide" :url='settingSlideUrl' :title="settingSlideTitle" :footer="settingSlideFooter" :header="settingSlideHeader" :position="settingSlidePosition"
				:height="settingSlideHeight" :modal='modal' :width='settingSlideWidth'  @cancel="settingSlideCancel" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
			</el-slide>

			<el-dialog title='<%=rb.getString("TongBu")%>' :visible.sync="syncDialog" ref="syncParamDialog" class="gnbMonitorSyncDialogCls" 
				width="660" :close-on-click-modal="false"  @close='closeSyncDialog' append-to-body>
					<div class="flex-ctn" >
						 <div class="select-all-cls" style="padding-left: 10px;">
							<el-checkbox v-model="alarmSync"></el-checkbox> 
							<span><%=rb.getString("GaoJingGuanLi")%></span>
							<span class="gray-color">( <%=rb.getString("HuoDongGaoJing")%> )</span>
							
						</div> 
						 <div class="select-all-cls" style="padding-left: 10px;">
							<el-checkbox :indeterminate="colSel" v-model="colAll" @change="colAllChange"></el-checkbox> 
							<span style="margin-right:20px;"><%=rb.getString("JianCeCanShuMing")%></span> 
							<div class="link-font" @click="expandedSync.params = !expandedSync.params">
								<span v-if="!expandedSync.params"><%=rb.getString("ZhanKai")%></span>
								<span v-else><%=rb.getString("GuanBi")%></span> 
							</div> 
						</div> 
						<div v-show="expandedSync.params">
							<div>
								<div class="select-all-cls">
									<i :class="{'el-icon':true,'el-icon-close1':expandedSync.device,'el-icon-open':!expandedSync.device}" @click="expandedSync.device = !expandedSync.device"></i>
									<el-checkbox :indeterminate="form.device.length<deviceCol.length && form.device.length>0" v-model="deviceAll" @change="deviceAllChange"></el-checkbox> 
									<span><%=rb.getString("JiChuPeiZhi")%></span>
								</div>
								<el-checkbox-group v-show="expandedSync.device" class="col-group" v-model="form.device">
									<el-checkbox v-for="item in deviceCol" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
								</el-checkbox-group>
							</div>
							<div>
								<div class="select-all-cls">
									<i :class="{'el-icon':true,'el-icon-close1':expandedSync.others,'el-icon-open':!expandedSync.others}" @click="expandedSync.others = !expandedSync.others"></i>
									<el-checkbox :indeterminate="form.others.length<othersCol.length && form.others.length>0" v-model="othersAll" @change="othersAllChange"></el-checkbox> 
									<span><%=rb.getString("GaoJiPeiZhi")%></span>
								</div>
								<el-checkbox-group v-show="expandedSync.others" class="col-group" v-model="form.others">
									<el-checkbox v-for="item in othersCol" :label="item.code" :key="item.code" :style="{'margin-left': '20px'}" :disabled="item.disabled">{{item.label}}</el-checkbox>
								</el-checkbox-group>
							</div>
						</div>
					</div>
					<span slot="footer">
						<div>
							<el-button type="primary" @click="goSynchronize"><%=rb.getString("QueDing")%></el-button>
							<el-button @click="closeSyncDialog"><%=rb.getString("QuXiao")%></el-button>
						</div>
					</span>
			</el-dialog>
			<!-- 收集TR069消息报文 -->
			<el-dialog title="<%=rb.getString("QueRen")%>" top="30vh" width="550"
				:visible.sync="collectMessageShow" 
				:modal="false"
				:close-on-click-modal="false">
				<el-form :model="collectForm">
					<div>{{confirmTips}}</div>
					<el-form-item label='<%=rb.getString("ChiXuShiChang")%>' style="display: flex;align-items: center;margin: 5px 0px;">
						<el-select v-model="collectForm.collectInterval" placeholder="Select time" class="collect-select">
							<el-option label="05" value="05"></el-option>
							<el-option label="10" value="10"></el-option>
						</el-select>
						<div style="display: inline;padding: 5px;margin-left: -4px;border: 1px solid #e9e9e9;background: #F5F7FA;"><%=rb.getString("ANRFenZhong")%></div>
					</el-form-item>
					<span v-if="collectExisted">
						<span style="color: #B3B3B3;"><%=rb.getString("ShouJiBaoWenFuGaiTiShi")%> SN={{existedMsgSN}}. </span>
					</span>
				</el-form>
	
				<div slot="footer" style="text-align: right;">
					<el-button type="primary" @click="sendCollect"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="collectMessageShow = false"><%=rb.getString("QuXiao")%></el-button>
				</div>
			</el-dialog>
			<!-- 收集TR069消息报文详情 -->
			<el-dialog :visible.sync="collectInfoShow" top="10vh">
				<div slot="title">
					<span class="el-dialog__title"><%=rb.getString("XinXi")%></span>
					<i class="el-icon el-icon-operation-export" style="position: absolute;right: 45px;top: 9px;" @click="downloadMsg"></i>
				</div>
				<el-input v-model="collectContent" type="textarea" rows="25" readonly class="none-border"></el-input>
			</el-dialog>
		</div>
		<script>
			var gnbCollectInterval = null;
			var gnbMonitor = new Vue({
				el: '#gnodeb_ctn',
				data() {	
					var vm = this,ignores=[];

					return {
						ignores: ignores,
						siteEnable: supportTopoSite,

						dragCol: [
							'serial_number',
							'product',
							'module_type',
							'host_name',
							'PHYCELLID',
							'op_state',
							'halob_flag',
							'ue_count',
							'cell_ip',
							'software_version',
							'group_name',
                            'mac_address'
						],
						drag: false,
						sortGroupShow: false,
						colGroups: {
							device: [],
							cell: [
								{code: 'nr_cell_id',label: '<%=rb.getString("NRXiaoQuID")%>'},
								{code: 'PHYCELLID',label: '<%=rb.getString("PCI2")%>',disabled: true},
								{code: 'tac',label: '<%=rb.getString("TAC")%>'},
								{code: 'Band',label: '<%=rb.getString("PinDuan")%>'},
								{code: 'EARFCNULINUSE',label: '<%=rb.getString("NRPinDianShangXian")%>'},
								{code: 'EARFCNDLINUSE',label: '<%=rb.getString("NRPinDianXiaXian")%>'},
								{code: 'tx_power',label: '<%=rb.getString("CPETxPower")%>'},
                                {code: 'network_model', label: '<%=rb.getString("JiZhanZhiShi")%>'},
							],
							status: [
								{code: 'adminState',label: '<%=rb.getString("AdminZhuangTai")%>'},
								{code: 'op_state',label: '<%=rb.getString("gNBZhuangTai")%>',disabled: true},
								{code: 'halob_flag',label: '<%=rb.getString("HaloBKaiGuan")%>',disabled: true},
								{code: 'ue_count',label: '<%=rb.getString("UEShu")%>',disabled: true},
								{code: 'synStatus',label: '<%=rb.getString("TongBuZhuangTai")%>'},
								{code: 'multiPlmnEnable',label: '<%=rb.getString("MultiPLMNZhuangTai")%>'},
								{code: 'rf_status', label: '<%=rb.getString("ShePinKaiGuanZhuangTai")%>'},
                                {code: 'amf_status', label: 'AMF Status'},
							],
							network: [
                                {code: 'IPSEC_ADDR', label: '<%=rb.getString("IPSECDiZhi")%>'},
								{code: 'cell_ip',label: '<%=rb.getString("IPDiZhi")%>',disabled: true}
							],
                            location: [
                                {code: 'gps_longitude', label: '<%=rb.getString("GPSJingDu")%>'},
                                {code: 'gps_latitude', label: '<%=rb.getString("GPSWeiDu")%>'},
                                {code: 'gps_height', label: '<%=rb.getString("GPSGaoDu")%>'},
                            ]
						},
						colForm: {
							device: ['serial_number','product','module_type','host_name','software_version','group_name','mac_address','online_time','offline_time'],
							cell: ['PHYCELLID'],
							status: ['op_state','halob_flag','ue_count'], 
							network: ['cell_ip'],
                            location: []
						},
						colExpanded: {
							device: true,
							cell: true,
							status: true,
							network: true,
                            location: true
						},
						showProps: '${showCol}'.split(','),
						sortColumns: [],
						dragColumns: [
							// {code: 'serial_number',label: '<%=rb.getString("XiaoZhanBianMa")%>',disabled: true},
							{code: 'product',label: '<%=rb.getString("ChanPinLeiXingBiaoZhi")%>',disabled: true},
							{code: 'product_name',label: '<%=rb.getString("ChanPinMingCheng")%>'},
							{code: 'module_type',label: '<%=rb.getString("SheBeiXingHaoMing")%>',disabled: true},
							// {code: 'host_name',label: '<%=rb.getString("5GZhanDianMingCheng")%>',disabled: true},
							{code: 'gNBId',label: '<%=rb.getString("GnodebId")%>'},
							{code: 'firmware_version',label: '<%=rb.getString("YingJianBanBen")%>'},
							{code: 'software_version',label: '<%=rb.getString("RuanJianBanBen")%>',disabled: true},
							{code: 'up_time',label: '<%=rb.getString("YunXingShiJian")%>'},
							{code: 'first_online_time',label: '<%=rb.getString("DiYiCiLianJieShiJian")%>'},
							{code: 'LASTINFORMTIME',label: '<%=rb.getString("ShangCiLianJieShiJian")%>'},
							{code: 'online_time', label: '<%=rb.getString("JieRuShiJian")%>'},
							{code: 'offline_time', label: '<%=rb.getString("DuanKaiShiJian")%>'},
							{code: 'group_name',label: '<%=rb.getString("SheBeiZu")%>',disabled: true},
							{code: 'mac_address', label: 'MAC',disabled: true},
							{code: 'sub_station_name', label: '<%=rb.getString("ZhanZhiMingCheng")%>',width: 120},
							{code: 'rf_status', label: '<%=rb.getString("ShePinKaiGuanZhuangTai")%>'},
							{code: 'nr_cell_id',label: '<%=rb.getString("NRXiaoQuID")%>'},
							{code: 'PHYCELLID',label: '<%=rb.getString("PCI2")%>',disabled: true},
							{code: 'tac',label: '<%=rb.getString("TAC")%>'},
							{code: 'Band',label: '<%=rb.getString("PinDuan")%>'},
							{code: 'EARFCNULINUSE',label: '<%=rb.getString("NRPinDianShangXian")%>'},
							{code: 'EARFCNDLINUSE',label: '<%=rb.getString("NRPinDianXiaXian")%>'},
							{code: 'tx_power',label: '<%=rb.getString("CPETxPower")%>'},
							{code: 'network_model', label: '<%=rb.getString("JiZhanZhiShi")%>'},
							{code: 'adminState',label: '<%=rb.getString("AdminZhuangTai")%>'},
							{code: 'op_state',label: '<%=rb.getString("gNBZhuangTai")%>',disabled: true},
							{code: 'halob_flag',label: '<%=rb.getString("HaloBKaiGuan")%>',disabled: true},
							{code: 'ue_count',label: '<%=rb.getString("UEShu")%>',disabled: true},
							{code: 'synStatus',label: '<%=rb.getString("TongBuZhuangTai")%>'},
							{code: 'multiPlmnEnable',label: '<%=rb.getString("MultiPLMNZhuangTai")%>'},
							{code: 'amf_status', label: 'AMF Status'},
							{code: 'IPSEC_ADDR', label: '<%=rb.getString("IPSECDiZhi")%>'},
							{code: 'gps_longitude', label: '<%=rb.getString("GPSJingDu")%>'},
							{code: 'gps_latitude', label: '<%=rb.getString("GPSWeiDu")%>'},
                            {code: 'gps_height', label: '<%=rb.getString("GPSGaoDu")%>'},
							{code: 'cell_ip',label: '<%=rb.getString("IPDiZhi")%>',disabled: true},
							{code: 'remark',label: 'Remark'}
						],

						exportColForm: {
							device: ['serial_number','product','module_type','host_name','software_version','group_name','mac_address','online_time','offline_time'],
							cell: ['PHYCELLID'],
							status: ['op_state','halob_flag','ue_count'],
							network: ['cell_ip'],
                            location: []
						},

						syncDialog:false,
                        syncType:'',
						alarmSync:true,
						othersCol: [
							{code: 'rollback_version', label: '<%=rb.getString("HuiTuiBanBen")%>'},
							{code: 'sas_param', label: 'SAS <%=rb.getString("JianCeCanShuMing")%>'},
							{code: 'eu_ru', label: 'EU/<%=rb.getString("RUShu")%>'},
							{code: 'halob_license', label: 'HaloB License'},
							{code: 'energy_saving', label: 'Energy'},
							{code: 'gnb_topo_cellmgr', label: 'gNB TOPO'},
							{code: 'ssl_cert_validity', label: 'SSL Cert Validity'},
						],
						form: {
							device: ['cell_name','IP','module_type','software_version','halob_flag','ue_count'],
							others:[]
						},
						expandedSync: {
							params:true,
							device: true,
							others:true,
						},
						rowData: {},

						expanded: false,
						slideTitle: '',
						slideURL: '',
						slidePosition: 'top',
					    sliderHeight: '100%',
					    sliderWidth: '75%',
                        modal: false,
						tbURL: '${ctx}/cell/cpeinfos/queryCpeInfosList.action',
						search_text:'',
						queryParams: {
							search_text: '',
							monitor: 1,
							TimeZone: timeZone,
							like_fields: 'serial_number,host_name,cell_ip',
							isDual: false,
							isMonitor: true,
							isGnb: 1,
							op_state: '',
							connection_status: '',
							software_version: '',
							firmware_version: '',
							group_id: '',
							product_model:'',
							model_name: '',
							halob_flag: '',
							multiPlmnEnable: ''
						},
						menus: [],
						topoCellManageSlide: {
	                        title: '',
	                        url: ''
	                    },
						settingSlideUrl:'',
						settingSlideTitle:'',
						settingSlideFooter:'',
						settingSlideHeader:'',
						settingSlidePosition:'',
						settingSlideHeight:'',
						settingSlideWidth:'',
						placeholderText:'<%=rb.getString("QingShuRu")%>',
						advancedQueryItemList:[
							{
								type:'checkbox',
								isShow:true,
								isIndeterminate:false,
								checkAll:false,
								popoverShow:false,
								checkedItemList:[],
								oldCheckedItemList:[],
								label:'<%=rb.getString("ZaiXianZhuangTai") %>',
								options:[
									{label:'<%=rb.getString("LianJieZhengChang")%>',value:"1"},
									{label:'<%=rb.getString("LianJieDuanKai")%>',value:"0"},
									{label:'<%=rb.getString("TongBuZhong")%>',value:"3"},
									{label:'<%=rb.getString("TongBuShiBai")%>',value:"2"}
								],
								value:'connection_status',
							},
							{
								type:'select',
								isShow:true,
								popoverShow:false,
								selectVal:'',
								label:'<%=rb.getString("gNBZhuangTai") %>',
								options:[
									{label:'<%=rb.getString("QuanBu")%>',value:''},
									{label:'<%=rb.getString("JiHuo")%>',value:"1"},
									{label:'<%=rb.getString("QuJiHuo")%>',value:"0"}
								],
								value:'op_state',
							},
							{
								type:'select',
								isShow:true,
								popoverShow:false,
								selectVal:'',
								label:'<%=rb.getString("ChanPinLeiXingBiaoZhi") %>',
								options:[
									{label:'<%=rb.getString("QuanBu")%>',value:''},
									{label:'BaiBNX',value:'BaiBNX'},
									{label:'BaiBNQ',value:'BaiBNQ'},
								],
								value:'product_model',
							},
							
							{
								type:'checkbox',
								isShow:false,
								isIndeterminate:false,
								checkAll:false,
								popoverShow:false,
								checkedItemList:[],
								oldCheckedItemList:[],
								label:'<%=rb.getString("SheBeiXingHaoMing") %>',
								options:[],
								value:'model_name',
							},
							{
								type:'checkbox',
								isShow:false,
								isIndeterminate:false,
								checkAll:false,
								popoverShow:false,
								checkedItemList:[],
								oldCheckedItemList:[],
								label:'<%=rb.getString("SoftwareVersion") %>',
								options:[],
								value:'software_version',
							},
							{
								type:'checkbox',
								isShow:false,
								isIndeterminate:false,
								checkAll:false,
								popoverShow:false,
								checkedItemList:[],
								oldCheckedItemList:[],
								label:'<%=rb.getString("YingJianBanBen") %>',
								options:[],
								value:'firmware_version',
							},
							{
								type:'checkbox',
								isShow:false,
								isIndeterminate:false,
								checkAll:false,
								popoverShow:false,
								checkedItemList:[],
								oldCheckedItemList:[],
								label:'<%=rb.getString("SheBeiZu") %>',
								options:[],
								value:'group_id',
							},
							{
								type:'select',
								isShow:false,
								isIndeterminate:false,
								checkAll:false,
								popoverShow:false,
								checkedItemList:[],
								oldCheckedItemList:[],
								label:'<%=rb.getString("MultiPLMNZhuangTai") %>',
								options:[
									{label:'<%=rb.getString("QuanBu")%>',value:''},
									{label:'<%=rb.getString("QiYong")%>',value:"1"},
									{label:'<%=rb.getString("JinYong")%>',value:"0"}
								],
								value:'multiPlmnEnable',
							},
							{
								type:'select',
								isShow:false,
								isIndeterminate:false,
								checkAll:false,
								popoverShow:false,
								checkedItemList:[],
								oldCheckedItemList:[],
								label:'<%=rb.getString("HaloBKaiGuan") %>',
								options:[
									{label:'<%=rb.getString("QuanBu")%>',value:''},
									{label:'<%=rb.getString("Kai")%>',value:"1"},
									{label:'<%=rb.getString("Guan")%>',value:"0"}
								],
								value:'halob_flag',
							},
							{
								type:'filter',
								popoverShow:false,
								checkedItemList:['connection_status','op_state','product_model'],
								oldCheckedItemList:['connection_status','op_state','product_model'],
								label:'<%=rb.getString("TianJiaShuaiXuan") %>',
								options:[
									{label:'<%=rb.getString("ZaiXianZhuangTai") %>',value:"connection_status"},
									{label:'<%=rb.getString("gNBZhuangTai")%>',value:"op_state"},
									{label:'<%=rb.getString("ChanPinLeiXingBiaoZhi")%>',value:"product_model"},
									{label:'<%=rb.getString("SheBeiXingHaoMing")%>',value:"model_name"},
									{label:'<%=rb.getString("SoftwareVersion")%>',value:"software_version"},
									{label:'<%=rb.getString("YingJianBanBen")%>',value:"firmware_version"},
									{label:'<%=rb.getString("SheBeiZu")%>',value:"group_id"},
									{label:'<%=rb.getString("MultiPLMNZhuangTai")%>',value:"multiPlmnEnable"},
									{label:'<%=rb.getString("HaloBKaiGuan")%>',value:"halob_flag"},
								],
								value:'add_filter',
							}

						],
						selectedRows: [],
						bulkSelectShow:false,

						collectExisted: false,
						existedMsgSN: '',
						collectMessageShow: false,
						collectForm: {
							collectInterval: ''
						},
						isGnbCollectExisted: false,
						collectDeviceCode: '',
						collectActiveSn: '',

						collectTaskTime: '',
						msgExtend: false,
						collectInfoShow: false,
						collectContent: '',
                        isExportDeviceLicenseInfo: false,

						percentage: '',
						exportFormatType: 'xlsx',
						
						editingRemarkLabel: false,
						remarkLabelInput: 'Remark',
						currentRemarkLabel: 'Remark',
					};
				},
				watch:{
					selectedRows(newVal){
						if(newVal.length == 0){
							this.bulkSelectShow = false;
						}
					},
					colGpDevice: {
						handler(newVal) {
							var vm = this;
							// 当 colGpDevice 计算属性变化时（比如 currentRemarkLabel 变化），更新 colGroups.device
							vm.colGroups.device = newVal;
						},
						deep: true
					}
				},
				computed: {
					deviceCol() {
						var vm = this;
						var cols = [
							{code: 'cell_name', label: '<%=rb.getString("5GZhanDianMingCheng")%>'},
							{code: 'IP', label: '<%=rb.getString("IPDiZhi")%>'},
							{code: 'module_type', label: '<%=rb.getString("SheBeiXingHaoMing")%>'},
							{code: 'software_version', label: '<%=rb.getString("RuanJianBanBen")%>'},
							{code: 'firmware_version', label: '<%=rb.getString("YingJianBanBen")%>'},
							{code: 'halob_flag', label: '<%=rb.getString("HaloBKaiGuan")%>'},
							{code: 'ue_count', label: '<%=rb.getString("UEShu")%>'},
							{code: 'sync_status',label: '<%=rb.getString("TongBuZhuangTai")%>'},
							{code: 'adminState', label: '<%=rb.getString("AdminZhuangTai")%>'},
							{code: 'amf_status', label: 'AMF Status'},
							{code: 'multiPlmnEnable', label: '<%=rb.getString("MultiPLMNZhuangTai")%>'},
							{code: 'ECI', label: 'ECI (<%=rb.getString("GnodebId")%> + Cell Identity)'},
							{code: 'cellConfig', label: '<%=rb.getString("XiaoQuCanShu")%>'},
							{code: 'MAC', label: 'MAC'},
							{code: 'mme_addr', label: '<%=rb.getString("IPSECDiZhi")%>'},
							{code: 'gps_position', label: '<%=rb.getString("GPSJingDu")%> + <%=rb.getString("GPSWeiDu")%>'}
						];
						
						if(supportTopoSite) {
							cols.push({code: 'sub_station_name', label: '<%=rb.getString("ZhanZhiMingCheng")%>'});
						}
						
						return cols.filter(function(item){ return !vm.ignores.includes(item.code);});
					},
					colGpDevice() {
						var vm = this;
						var cols = [
							{code: 'serial_number',label: '<%=rb.getString("XiaoZhanBianMa")%>',disabled: true},
							{code: 'product',label: '<%=rb.getString("ChanPinLeiXingBiaoZhi")%>',disabled: true},
							{code: 'product_name',label: '<%=rb.getString("ChanPinMingCheng")%>'},
							{code: 'module_type',label: '<%=rb.getString("SheBeiXingHaoMing")%>',disabled: true},
							{code: 'host_name',label: '<%=rb.getString("5GZhanDianMingCheng")%>',disabled: true},
							{code: 'gNBId',label: '<%=rb.getString("GnodebId")%>'},
							{code: 'firmware_version',label: '<%=rb.getString("YingJianBanBen")%>'},
							{code: 'software_version',label: '<%=rb.getString("RuanJianBanBen")%>',disabled: true},
							{code: 'up_time',label: '<%=rb.getString("YunXingShiJian")%>'},
							{code: 'first_online_time',label: '<%=rb.getString("DiYiCiLianJieShiJian")%>'},
							{code: 'LASTINFORMTIME',label: '<%=rb.getString("ShangCiLianJieShiJian")%>'},
							{code: 'online_time', label: '<%=rb.getString("JieRuShiJian")%>'},
							{code: 'offline_time', label: '<%=rb.getString("DuanKaiShiJian")%>'},
							{code: 'group_name',label: '<%=rb.getString("SheBeiZu")%>',disabled: true},
                            {code: 'mac_address', label: 'MAC',disabled: true},
                            // {code: 'authCode',label: '<%=rb.getString("JianQuanMa")%>',disabled: false},
                        ];
					
                        if(supportTopoSite) {
                            cols.push({code: 'sub_station_name', label: '<%=rb.getString("ZhanZhiMingCheng")%>'});
                        }
                        
                        // 确保 remark 始终在最后
                        cols.push({code: 'remark',label: vm.currentRemarkLabel ? vm.currentRemarkLabel : 'Remark'});
                        
                        return cols;
					},
					dragColumnsWithLabel() {
						var vm = this;
						// 返回带有动态label的dragColumns副本
						return vm.dragColumns.map(function(col) {
							if(col.code === 'remark') {
								return Object.assign({}, col, {
									label: vm.currentRemarkLabel ? vm.currentRemarkLabel : 'Remark'
								});
							}
							return col;
						});
					},
					limitBatch(){
						return batchOperation ? '' : 1;
					},
					progressHide() {
						return ['',null,undefined].includes(this.percentage);
					},
					dragOptions() {

						return {
							animation: 200,
							group: 'description',
							disabled: false,
							ghostClass: 'ghost'
						};
					},
					colGroupAll() {
						var vm = this;
						return vm.deviceGroupAll && vm.cellGroupAll && vm.statusGroupAll && vm.networkGroupAll && vm.locationGroupAll;
					},
					deviceGroupAll() {
						var vm = this;
						return vm.colGroups.device.length == vm.colForm.device.length;
					},
					cellGroupAll() {
						var vm = this;
						return vm.colGroups.cell.length == vm.colForm.cell.length;
					},
					statusGroupAll() {
						var vm = this;
						return vm.colGroups.status.length == vm.colForm.status.length;
					},
					networkGroupAll() {
						var vm = this;
						return vm.colGroups.network.length == vm.colForm.network.length;
					},
                    locationGroupAll() {
                        var vm = this;
                        return vm.colGroups.location.length == vm.colForm.location.length;
                    },
					showCols() {
						var vm = this;

						return vm.colForm.device.concat(vm.colForm.cell).concat(vm.colForm.status).concat(vm.colForm.network).concat(vm.colForm.location);
					},
					showColums() {
						var vm = this,
							columns = vm.dragColumns,
							props = columns.map(function(col){
								return col.code;
							});

						props = props.filter(function(code){
							return vm.showProps.includes(code);
						});

						return props;
					},

					exportcolGroupAll() {
						var vm = this;
						return vm.exportdeviceGroupAll && vm.exportcellGroupAll && vm.exportstatusGroupAll && vm.exportnetworkGroupAll && vm.exportlocationGroupAll;
					},
					exportdeviceGroupAll() {
						var vm = this;
						return vm.colGroups.device.length == vm.exportColForm.device.length;
					},
					exportcellGroupAll() {
						var vm = this;
						return vm.colGroups.cell.length == vm.exportColForm.cell.length;
					},
					exportstatusGroupAll() {
						var vm = this;
						return vm.colGroups.status.length == vm.exportColForm.status.length;
					},
					exportnetworkGroupAll() {
						var vm = this;
						return vm.colGroups.network.length == vm.exportColForm.network.length;
					},
                    exportlocationGroupAll() {
						var vm = this;
						return vm.colGroups.location.length == vm.exportColForm.location.length;
					},

					colSel() {
					
						var vm = this;
						var deviceSel = vm.form.device.length<vm.deviceCol.length && vm.form.device.length>0 ;
						var otherSel = vm.form.others.length<vm.othersCol.length && vm.form.others.length>0 ;
							//两组全选
							if(vm.colAll){
								return false
							}else if(vm.form.device.length>0 || vm.form.others.length>0){ //半选 
								return true
							}else {
								return false;
							}
						
					},
					colAll () {
						var vm = this;
						return vm.deviceAll && vm.othersAll;
					},
					deviceAll() {
						var vm = this;
						return vm.deviceCol.length == vm.form.device.length;
					},
					othersAll() {
						var vm = this;
						return vm.othersCol.length == vm.form.others.length;
					},
					ctnCls() {
						var vm = this;
						
						return {
							'container': true,
							'expanded': vm.expanded
						}
					},
					isMonitorWritable() {
						return writableMap.CODE_GNB_MONITOR == true;
					},
					isAdmin() {
						return is_super_user == 'true'
					},
					confirmTips() {
						var vm = this,
							sn = vm.rowData.serial_number,
							msg = '<%=rb.getString("QueRenShouJiPre")%>';
	
						return msg.replace('placeholder', sn);
					},
				},
				methods: {
					init() {
						var vm = this;
						
						axios.post("${ctx}/cell/cpeinfos/getFirmwareVersionList.action?isGnb=1").then(function(response){
							var data = response.data ? response.data : [];
							var arr = [];
							data.map(function(item){
								arr.push({label:item.firmware_version,value:item.firmware_version})
							})
							vm.advancedQueryItemList.map((items)=>{
								if('firmware_version' == items.value){
									items.options = arr
								}
							})
						})
						axios.post("${ctx}/cell/cpeinfos/getModelNameList.action?isGnb=1").then(function(response){
							var data = response.data ? response.data : [];
							var arr = [];
							data.map(function(item){
								arr.push({label:item.module_type,value:item.module_type})
							})
							vm.advancedQueryItemList.map((items)=>{
								if('model_name' == items.value){
									items.options = arr || []
									// items.options.unshift({label:'<%=rb.getString("QuanBu")%>',value:''});
								}
							})
						})
						axios.post("${ctx}/cell/cpeinfos/getEnbMonitorProductList.action?isGnb=1").then(function(response){
							var data = response.data ? response.data : [];
							var arr = [];
							data.map(function(item){
								if (item){
									arr.push({label:item,value:item})
								}
							})
							vm.advancedQueryItemList.map((items)=>{
								if('product_model' == items.value){
									items.options = arr || []
									items.options.unshift({label:'<%=rb.getString("QuanBu")%>',value:''})
								}
							})
						})
						$.getJSON('${ctx}/cell/cpeinfos/getCellVersionList.action?isGnb=1',function(data){
							var data = data;
							var arr = [];
							data.map(function(item){
								arr.push({label:item.software_version,value:item.software_version})
							})
							vm.advancedQueryItemList.map((items)=>{
								if('software_version' == items.value){
									items.options = arr
								}
							})
						}); 
						
						$.getJSON('${ctx}/cell/cpeinfos/getDeviceGroupListByCell.action?isGnb=1',function(data){
							var data =data;
							var arr = [];
							data.map(function(item){
								if (item){
									arr.push({label:item.group_name,value:item.id})
								}
							})
							vm.advancedQueryItemList.map((items)=>{
								if('group_id' == items.value){
									items.options = arr;
								}
							})
						}) 
					},
					// 模糊搜索
					query() {
						var vm = this;
						vm.queryParams.search_text = this.search_text;
						vm.queryParams['like_fields'] = 'serial_number,host_name,cell_ip';
					},
					// 搜索域聚焦事件
					queryInputFocus(){
						var vm = this;
						vm.placeholderText = '<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("5GZhanDianMingCheng")%> / <%=rb.getString("IPDiZhi")%>';
					},
					// 搜索域失焦事件
					queryInputBlur(){
						var vm = this;
						vm.placeholderText = '<%=rb.getString("QingShuRu")%>';
					},
					// 高级查询 确定事件
					advanceQuery(type,paramsItem,value){
						var vm = this,
							params ={};
						if(type == 'select'){
							params[paramsItem] = value;
						}else{
							params[paramsItem] = value.join(',');
						}
						Object.assign(vm.queryParams, params);
					},
					queryProgress() {
						var vm = this,
							url = '${ctx}/gnb/gnbMonitor/getExportGNBCellsProgress.action';

						clearInterval(gnbProgressInterval);
						gnbProgressInterval = setInterval(function() {
							axios.post(url).then(function(res){
								var data = res.data;

								vm.percentage = data - 0;

								if(vm.percentage == 100 || data === '') {
									clearInterval(gnbProgressInterval);
									setTimeout(function(){
										vm.percentage = '';
									},1500)
								}
							});
						},1000);
					},
					// 高级搜索下拉全选事件
					handleCheckedAllChange(item){
						var vm = this,
							allList = [];
						item.options.map((items)=>{
							if(items.value){
								allList.push(items.value)
							}
						})
						vm.advancedQueryItemList.map((items)=>{
							if(item.value == items.value){
								items.checkedItemList = items.checkAll? allList : [];
								items.isIndeterminate = false;
							}
						})
					},
					// 高级搜索下拉单选事件
					handleCheckedChange(item){
						var vm = this,
							checkCount = item.checkedItemList.length,
							allCount = item.options.length;
						
						vm.advancedQueryItemList.map((items)=>{
							if(item.value == items.value){
								if(items.type == 'checkbox'){
									items.checkAll = checkCount === allCount;
									items.isIndeterminate = checkCount > 0 && checkCount < allCount;
								}
							}
						})
					},
					// 高级搜索下拉弹窗 展开事件
					checkPopoverShow(item){
						var vm = this;
						vm.advancedQueryItemList.map((items)=>{
							if(item.value == items.value){
								items.popoverShow = true;
							}
						})
					},
					// 高级搜索下拉弹窗 收起事件
					checkPopoverHide(item,types){
						var vm = this,
							checkCount = 0,
							allCount = item.options.length;
						if(item.type != 'select'){
							checkCount = item.oldCheckedItemList.length
						}
						vm.advancedQueryItemList.map((items)=>{
							if(item.value == items.value){
								items.popoverShow = false;
								if(items.type != 'select'){
									items.checkedItemList = items.oldCheckedItemList;
								}
								if(items.type == 'checkbox'){
									items.checkAll = checkCount === allCount;
									items.isIndeterminate = checkCount > 0 && checkCount < allCount;
								}
							}
						})
						if(types == 'del'){
							document.body.click();
						}
					},
					// 高级搜索下拉弹窗 选择提交
					checkPopoverSubmit(item){
						var vm = this,
							params ={};
						if(item.type == 'checkbox'){
							vm.advancedQueryItemList.map((items)=>{
								if(item.value == items.value){
									items.oldCheckedItemList = items.checkedItemList;
									if(items.checkAll){
										params[items.value] = '';
									}else{
										params[items.value] = items.checkedItemList.join(',');
									}
								}
							})
						}else{
							vm.advancedQueryItemList.map((items)=>{
								if(item.value == items.value){
									items.oldCheckedItemList = items.checkedItemList;
								}else{
									var result = item.checkedItemList.includes(items.value);
									if(!result){
										items.isShow = false;
										items.checkedItemList = [];
										items.oldCheckedItemList = [];
										items.checkAll = false;
										items.isIndeterminate = false;
										params[items.value] = '';
									}else{
										items.isShow = true;
									}
								}
							})
						}
						
						Object.assign(vm.queryParams, params);
						document.body.click();
					},
					// 筛选项删除事件
					checkItemDel(item){
						var vm = this,
							params ={};
						vm.advancedQueryItemList.map((items)=>{
							if(item.value == items.value){
								items.isShow = false;
								if(items.type != 'select'){
									items.checkedItemList = [];
									items.oldCheckedItemList = [];
									items.checkAll = false;
									items.isIndeterminate = false;
								}else{
									items.selectVal = '';
								}
								params[items.value] = '';
							}
							if(items.value == 'add_filter'){
								var idx = items.checkedItemList.indexOf(item.value),
									oldIdx = items.oldCheckedItemList.indexOf(item.value);
								if(idx>-1){
									items.checkedItemList.splice(idx,1)
								}
								if(oldIdx>-1){
									items.oldCheckedItemList.splice(idx,1)
								}
							}

						})
						Object.assign(vm.queryParams, params);
					},
					// //单选 选择事件
					selectChangeClick(value,item){
						var vm = this,
							params ={};
						
						vm.advancedQueryItemList.map((items)=>{
							if(item.value == items.value){
								items.selectVal = value;
								params[items.value] = value;
							}
						})
						Object.assign(vm.queryParams, params);
						document.body.click();
					},
					// 清除筛选
					clearFilterClick(){
						var vm = this,
							params = {};
						vm.advancedQueryItemList.map((items)=>{
							if(items.isShow && items.isShow== true ){
								if(items.type == 'checkbox'){
									items.checkedItemList = [];
									items.oldCheckedItemList = [];
									items.checkAll = false;
									items.isIndeterminate = false;
								}else if(items.type == 'select'){
									items.selectVal = '';
								}
								if(items.type == 'checkbox' || items.type == 'select'){
									params[items.value] = '';
								}
							}
						})
						Object.assign(vm.queryParams, params);
						document.body.click();
					},
					//告警数量点击
					alarmToInfo(row){
	                    var vm = this;
	                    vm.settingBtnClick(row,'alarm');
	                },
					columnSetting() {
						var vm = this;

						vm.sortGroupShow = !vm.sortGroupShow;
					},
	                //设置图标点击
					settingBtnClick(row,page){						
						var vm = this,str = Math.random().toString();
						
						// 检查 eNBTopo_tab.jsp 是否已打开 setting.jsp 页面，如果打开则先关闭以防止 ID 冲突
						if(typeof topovm !== 'undefined') {
							try {
								// 直接尝试关闭 eNBTopo_tab 的 slide，即使它没有打开也不会报错
								topovm.$refs.topoSettingSlide.hide();
							} catch(e) {}
						}
						
						vm.slideURL = '${ctx}/gnb/setting/openSettingPage.action?randomValue='+str;
						vm.slidePosition = 'top';
		    	    	vm.sliderHeight = '100%';
		    	    	vm.sliderWidth = '72%';
		    	    	vm.$refs.slide.showSlide(function(){
		    	    		eventBus.$emit('action-settingPage', row, page, 'monitor');
		    	    	});
					},
					operClick(evt,row) {
						var vm = this,
							opState = row.op_state,
							connectStatus = row.connection_status,
							DeviceLogView = "${DeviceLogView}",
							opStateName = '',
							opSwitch = '',
							rfStatus = row.rf_status||'',
							tongbuDisableFlag = true,
							enbActiveFlag = true,
							opStateFlag = true,
							rfDisableFlag = true
							chongQidisableFlag = true,
							rizhidisableflag = true,
							dualCarrierType = row.dual_carrier_type,
                        	collectShow = dualCarrierType != 2;
						vm.rowData = row;

						if(opState == "1"){
							opStateName = '<%=rb.getString("goJiHuo")%>';
							opSwitch = "0";
						}else{
							opStateName ='<%=rb.getString("JiHuo")%>';
							opSwitch = "1";
						}
						//配置恢复 判断
						var resetDisableFlag;
						
						if(connectStatus != 'Off'){
							tongbuDisableFlag = false;
							opStateFlag = false;
							chongQidisableFlag = false;
							resetDisableFlag = false;
							rfDisableFlag = false;
						}else{
							tongbuDisableFlag = true;
							opStateFlag = true;
							chongQidisableFlag = true;
							resetDisableFlag = true;
							rfDisableFlag = true;
						}
						//判断halob菜单项是否显示  可用
						//当基站不在线  且不是集中模式   不显示但      非集中模式 且基站在线 菜单可用     非集中模式 且基站不在线 菜单不可用
						var halobName = "",
							disableHalobFlag,
							halobSwitchShow,
							halobSwitch;
						if(connectStatus != 'Off') disableHalobFlag = false;
						else disableHalobFlag = true;
						
						$.ajax({
							type:'POST',
							url:'${ctx}/cell/cpeinfos/checkCellHasLicenseInfo.action',
							data:{small_cell_code:row.small_cell_code},
							async:false,
							dataType:'json',
							success:function(data){
								
								//控制halob菜单
								if([0,'0'].includes(data["halobFlag"])){
									//关闭Halob
									halobName = '<%=rb.getString("KaiQiHaloB")%>';
									halobSwitch = 1;
									halobSwitchShow = true;
								}else if(data["halobFlag"] == 1){
									//开启Halob
									halobSwitch = 0;
									halobSwitchShow = true;
									halobName = '<%=rb.getString("GuanBiHaloB")%>';
								}else{
									//集中模式   集中模式不显示 halob的菜单
									halobSwitchShow = false;
									halobName = '<%=rb.getString("HaloBKaiGuan")%>';
								}
							},
						});

						if(DeviceLogView != '0'){
							if(connectStatus != 'Off'){
								rizhidisableflag = false
							}else{
								rizhidisableflag = true;
							}
							
						} else{
							rizhishowflag = false;
						}

						var activeChild = [];
                        
						opState.split(',').map(function(s, idx) {
							var opIdxName = 'Cell' + (idx+1) + '  <%=rb.getString("JiHuo")%>',
								opIdxSwitch = '1',
								opIndex = '41'+idx;
							
							if(s == "1"){
								opIdxName = 'Cell' + (idx+1) + '  <%=rb.getString("goJiHuo")%>';
								opIdxSwitch = "0";
							}

							activeChild.push({
								code: 'multActive',
								row: row, 
								label: opIdxName, 
								id: (opIndex-0), 
								cls: '',
								show: true,
								disable: opStateFlag,
								activeStatus: opIdxSwitch,
								cell_code: row.small_cell_code, 
								cellNumber: (idx+1)
							});
						});
						var rfText,
							rfChild = [],
							showCell = false,
							rfStatusShowfloag = true;
						
						// 根据RF开关状态，为操作显示的文字赋值 
						if(rfStatus === '' || rfStatus =='--'){
							rfStatusShowfloag = false;
						}else if ( rfStatus == "on" || rfStatus == 1){
							showCell = false;
							rfText = '<%=rb.getString("ShePinCaoZuo")%>' + ' '+ '<%=rb.getString("Guan")%>';
						}else if( rfStatus == "off" || rfStatus === 0){
							showCell = false;
							rfText = '<%=rb.getString("ShePinCaoZuo")%>' + ' '+ '<%=rb.getString("Kai")%>';
						}else{
							showCell = true;
							rfText = '<%=rb.getString("ShePinCaoZuo")%>';
							rfStatus = rfStatus.split(",");
	
							rfStatus.map(function(sItem, idx){
								var rfItemText = 'RF' + (idx+1) + ' ' + (sItem == 'on'? '<%=rb.getString("Guan")%>' : '<%=rb.getString("Kai")%>'),
									rfIndex = '43'+idx;;
								
								rfChild.push({
									code: 'multRf',
									row: row, 
									label: rfItemText,
									id: (rfIndex-0),
									cls: ' ',
									rfStatus: sItem,
									cell_code: row.small_cell_code,
									show: true,
									cellNumber: idx+1
								});
							})
						}

						vm.menus = [
							{ row: row,cls:'', show: writableMap['CODE_GNB_SYNCHRONIZE'] == true || (writableMap['CODE_GNB_TR069_MSG_EXCHANGE'] == true && collectShow),
								child: [
									{code: 'sync', label: '<%=rb.getString("TongBu")%>',row: row,disable: tongbuDisableFlag, show: writableMap['CODE_GNB_SYNCHRONIZE'] == true},
									{code: 'collect',row: row, label:'<%=rb.getString("ShouJiBaoWen")%>',show: collectShow && writableMap['CODE_GNB_TR069_MSG_EXCHANGE'] == true},
								]
							},
							{ row: row,cls:'', show: writableMap['CODE_GNB_REBOOT'] == true,
								child: [
									{code: 'reboot', label: '<%=rb.getString("ChongQi")%>',row: row,disable: chongQidisableFlag, show: writableMap['CODE_GNB_REBOOT'] == true},
								]
							},
							{ row: row,cls:'', show: writableMap['CODE_GNB_ACTIVE'] == true,
								child: activeChild
							},
							{ row: row,cls:'', show: writableMap['CODE_GNB_RF_ENABLE'] == true && rfStatusShowfloag && !showCell,
								child: [
									{code: 'multRf', label: rfText,row: row,disable: rfDisableFlag},
								]
							},
							{ row: row,cls:'', show: writableMap['CODE_GNB_RF_ENABLE'] == true && rfStatusShowfloag && showCell,
								child: rfChild
							},
							{ row: row,cls:'', show: halobSwitchShow,
								child: [
								    {code: 'halobEnable',label:halobName,row: row,cls:'', disable: disableHalobFlag,halob_switch:halobSwitch},
								]
							},
							{ row: row,cls:'', show: writableMap['CODE_GNB_LOGS'] == true,
								child: [
									{code: 'logs', label: '<%=rb.getString("RiZhi")%>',row: row,disable: rizhidisableflag },
								]
							},
						];

						vm.showMenus(evt);
					},
					hideMenus() {
						this.$refs.menu.hide()
					},
					showMenus(evt) {
						this.$refs.menu.show(evt)
					},
					menuClick(menu) {
						var vm = this,
							codes = {
								//setting: vm.goSetting,
								sync: vm.openSyncDialog,
								active: vm.goActive,
								multActive: vm.goMultActive,
								reboot: vm.goReboot,
								reset: vm.goReset,
								logs: vm.goLogs,
								halobEnable:vm.openCloseHalob,
								collect:vm.showCollectMessage,
								multRf:vm.setRFStatus
							},
							code = menu.code;

						select_row_data = menu.row;
						
						if(codes[code]) {
							if(menu.code == 'halobEnable'){
								codes[code](menu.row,menu.halob_switch)
							}else if(menu.code == 'multActive' || menu.code == 'multRf') {
								codes[code](menu)
							}else{
								codes[code](menu.row)
							}
						}
					},
					goMultActive(menuRow) {
						var vm = this;

						var params = {
								op_state: menuRow.activeStatus,
								small_cell_code: menuRow.cell_code,
								cellNumber: menuRow.cellNumber
							};
						
						$.post("${ctx}/cell/cpeinfos/cellModifyActiveStatus.action",params,function(data){
							if(data["success"]){
								vm.$refs.monitor.refresh();
								showMsg('success_msg','<%=rb.getString("XiaFaChengGong")%>');
							}else{
								showMsg('error_msg',data["message"]);
							}
						},"json")
					},
					setRFStatus(menuRow){
						var vm = this,
							status = menuRow.row.rf_status,
							params = {
								smallCellCode: menuRow.row.small_cell_code,
								radioStatus: status,
								cellNumber: menuRow.cellNumber ? menuRow.cellNumber : '1',
							};
						// 依据状态设置开、关
						if(status == 'on') params.radioStatus = 'off';
						if(status == 'off') params.radioStatus = 'on';
						
						$.ajax({
							url: '${ctx}/cell/cpeinfos/cellModifyRadioStatus.action',
							data: params,
							type: 'post',
							dataType: 'json',
							success: function(data){
								if(data.success){
									vm.$refs.monitor.refresh();
								}
								showMsg('success_msg','<%=rb.getString("XiaFaChengGong")%>');
							},
							error: function(data){
								showMsg('error_msg',data["message"]);
							}
						});
					},
					hideSlide(){
						this.$refs.slide.hide();
                        this.$refs.monitor.refresh();
					},
					goSetting(row) {
						var vm = this,
							str = Math.random().toString();

						vm.settingSlideUrl = '${ctx}/cell/quicksettings/goGNBQuickSettingPage.action?randomValue='+str;
						vm.settingSlideHeight = '100%';
						vm.settingSlideWidth = '100%';
						vm.settingSlideFooter = false;
						vm.settingSlidePosition = 'top';
						vm.settingSlideHeader = false;
						vm.settingSlideTitle = '';
						vm.$refs.settingSlide.showSlide(function(){
							eventBus.$emit('gnbSetting-init',row);
						});
					},
					openSyncDialog(){
                        var vm = this;
                        vm.syncType = 'single';
						vm.syncDialog = true;
					},
					closeSyncDialog(){
						var vm = this;
						vm.syncDialog = false;
						vm.form.device = [];
						vm.form.others = [];
						vm.alarmSync = true;
					},
					colAllChange(val) {
						var vm = this;
		
						vm.deviceAllChange(val);
						vm.othersAllChange(val);
					},
					deviceAllChange(val) {
						var vm = this,
							fields = vm.deviceCol.map(function(item){
								return item.code;
							}),
							filters = vm.deviceCol.filter(function(item){
								return item.disabled;
							}).map(function(item){
								return item.code;
							});
						
						vm.form.device = val?fields:filters;
					},
					othersAllChange(val) {
						var vm = this,
							fields = vm.othersCol.map(function(item){
								return item.code;
							}),
							filters = vm.othersCol.filter(function(item){
								return item.disabled;
							}).map(function(item){
								return item.code;
							});
						
						vm.form.others = val?fields:filters;
					},
					goSynchronize(row) {
						var vm = this,
                            url = '',
							params = {
								smallCellCode: '',
								isGnb: 1,
								selectedParams:''
							};

						var content ='';
						var selectArr = [];
						selectArr.push(vm.form.device);
						selectArr.push(vm.form.others);
										
						for(var i=0;i<selectArr.length;i++){
							if(selectArr[i].length>0){
								content += ','+selectArr[i].join(',');
							}
						}
						if(vm.alarmSync) {
							content += ',sync_alarm';
						}
						content = content.substring(1);
								
						params.selectedParams = content;
                        if(vm.syncType == 'batch'){
                            var cellCodes = vm.selectedRows.map(function(item){
                                return item.small_cell_code
                            }).join(',');
                            params.smallCellCode = cellCodes;
                            url="${ctx}/cell/quicksettings/batchSyncCell.action"
                        }else {
                            params.smallCellCode = vm.rowData.small_cell_code;
                            url="${ctx}/cell/param/refreshCellInfo.action"
                        }
						
						$.post( url, params, function(data){
							if (data["success"]) {
								vm.closeSyncDialog();
                                if(vm.syncType == 'batch'){
                                    vm.$refs.monitor.clearSelection();
                                }
							} else {
								showMsg('error_msg',data["message"]);
							}
						}, "json");
					},
					goActive(row) {
						var vm = this,
							params = {
								op_state: row.op_state?0:1,
								small_cell_code: row.small_cell_code
							};

						$.post("${ctx}/cell/cpeinfos/cellModifyActiveStatus.action",params,function(data){
							if(data["success"]){
								vm.$refs.monitor.refresh();
							}else{
								showMsg('error_msg',data["message"]);
							}
						},"json")
					},
					goReboot(row) {
						var vm = this,
							cell_code = row.small_cell_code;
						
						$.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingChongQiSheBei")%>", function (r) {
							if (r) {
								showMsg('prompt_msg',"<%=rb.getString("MingLingYiXiaFa")%>")
								var param = {
									cell_code: cell_code,
									isGnb: 1
								};
								$.post("${ctx}/cell/cpeinfos/cellReboot.action", param, function (data) {
									if (!data["success"]) {
										showMsg('error_msg',data["message"]);
									}
								}, "json");
							}
						}).addClass("seriousConfirm");
					},
					goReset(row) {
						var vm = this,
							code = row.small_cell_code;
						
						vm.$confirm('<%=rb.getString("QueDingHuiFuMoRenPeiZhi")%>','<%=rb.getString("QueRen")%>').then(function(r){
							if (r) {
								var param = {
										cellCode: code,
										isGnb: 1
									};
								
								axios.post("${ctx}/cell/cpeinfos/cellFactoryReset.action", stringify(param)).then(function(res){
									var data = res.data;
									
									if(data.success == true) {
										vm.$message({
				    						message: '<%=rb.getString("MingLingYiXiaFa")%>',
				    						type: 'success'
				    					});
									}else {
										vm.$message.error(data["message"]);
									}
								});
							}
						}).catch(function(){});
					},
					goLogs(row) {
						var vm = this,
							cellCode = row.small_cell_code,
						    serial_number = row.serial_number,
							param = {
								start_time: 'undefined',
								end_time: 'undefined',
								execute_type: 'Immediately',
								reportPeriod: '',
								isReboot: 'false',
								serial_number: serial_number, 				    				
								timeZone: timeZone, 
								device_type: "eNB",
								device_code: cellCode,
								isGnb: 1
							};

						$.post("${ctx}/cell/collect/goImmediateCollectLogFile.action", param, function (data) {
							if (data["success"]) {
								showMsg('success_msg','<%=rb.getString("gNBRiZhiShouJiTiShi")%>')
							} else {
								showMsg('error_msg',data['message']);
							}
						}, "json");
					},
					openCloseHalob(row,halobSwitch){
						var vm = this,
							params = {};
						params.cell_code = row.small_cell_code;
						params.halob_switch = halobSwitch;
						vm.$confirm('Halob Switch needs reboot gNB.Are you sure to modify?','<%=rb.getString("QueRen")%>',{
							customClass:'warningConfirm',
							confirmButtonText:'<%=rb.getString("QueDing")%>',
							cancelButtonText:'<%=rb.getString("QuXiao")%>',
							type:'warning',
							closeOnClickModal:false
						}).then(() => {
							$.post("${ctx}/cell/cpeinfos/setCellHalobSwitch.action",params,function(data){
								if(data["success"]){
									
								}
							},"json")
						}).catch(() => {
							
						})
					},
					// 打开收集报文弹窗
					showCollectMessage(row) {
						var vm = this,
							paramsExist = {
								type: 'gnb',
								operatorCode: operatorCodeGloab
							};
	
						axios.post('${ctx}/trace/isExistTracingDevice.action', stringify(paramsExist)).then(function(res){
							var data = res.data;
	
							if(data && data.isExist == 'true') {
								vm.collectExisted = true;
								vm.existedMsgSN = data.serialNumber;
							}else {
								vm.collectExisted = false;
								vm.existedMsgSN = '';
							}
						});
	
						vm.collectForm.collectInterval = '10';
						vm.collectMessageShow = true;
					},
					// 确认收集报文
					sendCollect() {
						var vm = this,
							row = vm.rowData || {},
							time = vm.collectForm.collectInterval+':00',
							params = {
								deviceCode: row.small_cell_code,
								serialNumber: row.serial_number,
								type: 'gnb',
								operatorCode: operatorCodeGloab,
								collectInterval: vm.collectForm.collectInterval
							},
							paramsExist = {
								type: 'gnb',
								operatorCode: operatorCodeGloab
							};
	
						axios.post('${ctx}/trace/isExistTracingDevice.action', stringify(paramsExist)).then(function(res){
							var data = res.data;
	
							if(data && data.isExist == 'true') {
								vm.$message.error('SN=' + data.serialNumber + '<%=rb.getString("ZhengZaiShouJi")%>');
							}else {
								axios.post('${ctx}/trace/start.action', stringify(params)).then(function(res){
									var data = res.data;
	
									if(data.success == true) {
										vm.startCollectInterval(time);
										vm.queryLatestCollectInfo();
										vm.collectMessageShow = false;
										
										vm.$message.success('<%=rb.getString("ChengGong")%>');
									}else {
										vm.$message.error(data.message);
									}
								});
							}
						});
					},
					// 查询最新收集报文详情
					queryLatestCollectInfo() {
						var vm = this,
							params = {
								type: 'gnb',
								operatorCode: operatorCodeGloab
							};
	
						axios.post('${ctx}/trace/queryLatestMessageTraceDeviceInfo.action', stringify(params)).then(function(res){
							var data = res.data;
	
							if(data) {
								vm.collectActiveSn = data.serialNumber;
								vm.collectDeviceCode = data.deviceCode;
	
								if(data.status == '0' && data.remainTime) {
									vm.startCollectInterval(data.remainTime);
								}else if(data.status == '1'){
									vm.collectTaskTime = '';
									vm.startCollectInterval('00:01');
								}
	
								if(['',undefined].includes(data.status) && ['',undefined].includes(data.serialNumber)) {
									vm.isGnbCollectExisted = false;
								}else {
									vm.isGnbCollectExisted = true;
								}
							}
						});
					},
					// 查看收集报文详情
					viewMsg() {
						var vm = this,
							params = {
								serialNumber: vm.collectActiveSn,
								type: 'gnb'
							};
							   
						vm.collectInfoShow = true;
	
						axios.post('${ctx}/trace/queryMessageTraceInfo.action', stringify(params)).then(function(res){
							var data = res.data;
	
							if(data) {
								vm.collectContent = data;
								vm.collectInfoShow = true;
							}
						});
					},
					// 下载收集报文
					downloadMsg() {
						var vm = this,
							params = {
								serialNumber: vm.collectActiveSn,
								type: 'gnb',
								timeZone: timeZone
							};
	
						exportByForm('${ctx}/trace/downLoadMessageTraceInfo.action',params);
					},
					// 清除收集报文任务
					clearMsg() {
						var vm = this,
							params = {
								deviceCode: vm.collectDeviceCode,
								serialNumber: vm.collectActiveSn,
								type: 'gnb',
								operatorCode: operatorCodeGloab
							};
	
						axios.post('${ctx}/trace/clear.action', stringify(params)).then(function(res){
							var data = res.data;
	
							if(data.success == true) {
								vm.$message.success('<%=rb.getString("ChengGong")%>');
								vm.queryLatestCollectInfo();
							}else {
								vm.$message.error(data.message);
							}
						});
					},
					// 停止收集报文
					stopCollect() {
						var vm = this,
							params = {
								deviceCode: vm.collectDeviceCode,
								serialNumber: vm.collectActiveSn,
								type: 'gnb',
								operatorCode: operatorCodeGloab
							};
	
						axios.post('${ctx}/trace/stop.action', stringify(params)).then(function(res){
							var data = res.data;
	
							if(data.success == true) {
								vm.$message.success('<%=rb.getString("ChengGong")%>');
								vm.queryLatestCollectInfo();
							}else {
								vm.$message.error(data.message);
							}
						});
					},
					startCollectInterval(time) {
						var vm = this,
							list = time.split(':'),
							totalTime = list[0]*60 + list[1]*1;
	
						clearInterval(gnbCollectInterval);
						
						gnbCollectInterval = setInterval(function() {
							totalTime -= 1;
							vm.collectTaskTime = vm.formaterTime(Math.floor(totalTime/60)) + ':' + vm.formaterTime(totalTime%60);
	
							if(totalTime<1) {
								clearInterval(gnbCollectInterval);
								vm.collectTaskTime = '';
							}
						},1000);
					},
					formaterTime(num) {
	
						return num < 10? '0'+num : num;
					},
					exportForm() {
						var vm = this,
							codes = vm.exportColForm.device.concat(vm.exportColForm.cell).concat(vm.exportColForm.status).concat(vm.exportColForm.network).concat(vm.exportColForm.location);
							params = {
								content: codes.join(',')
							};

						//params.content = params.content.replace('lastsyntime', 'LASTINFORMTIME');
						var url = '${ctx}/gnb/gnbMonitor/exportGnbInfoToExcel.action';
						if(vm.exportFormatType == 'csv') {
							url = '${ctx}/gnb/gnbMonitor/exportGnbInfoToCsv.action';
						}

						Object.assign(params, vm.queryParams);

						downLoadFileByAxios(url, params);
                        setTimeout(function(){
                            if(vm.isExportDeviceLicenseInfo){
                                params.isGnb = 1;
                                downLoadFileByAxios('${ctx}/cell/cpeinfos/exportEnodebLicenseInfos.action', params);
                                vm.isExportDeviceLicenseInfo = false;
                            }
                        },1000);

						if(vm.exportFormatType != 'csv') vm.queryProgress();

						document.body.click();
					},
					settingSlideCancel(){
						var vm =this;
						vm.$refs.monitor.refresh();
						vm.$refs.settingSlide.hide();
					},
					syncName(code, newName) {
	                    var vm = this;

	                    $.post('${ctx}/cell/cpeinfos/syncCellName.action?smallCellCode='+code, function(data){
	                        if(data.success){
	            				vm.refreshList();
	                            vm.closeSyncName();
	                        }else{
	                            showMsg('error_msg',data["message"]);
	                        }
	                    },'json'); 
	                },
					closeSyncName() {
	                    document.body.click();
	                },
					ueCountsFormatter(rowData, value, rowIndex) {

						if(value === -1 || value === null || value === '' || value === undefined){
							return "<a style='color:#000000;text-decoration:none;cursor:default;' href='#'>--</a>"; 
						}else{
							return value
						}
					},

					
					clickColumnLabel(label) {
						$('#gnodeb_ctn .el-checkbox__label:contains('+label+')').click();
					},
					colGroupAllChange(val) {
						var vm = this;

						vm.deviceGroupAllChange(val);
						vm.cellGroupAllChange(val);
						vm.statusGroupAllChange(val);
						vm.networkGroupAllChange(val);
                        vm.locationGroupAllChange(val);
					},
					deviceGroupAllChange(val) {
						var vm = this,
							fields = vm.colGroups.device.map(function(item){
								return item.code;
							}),
							filters = vm.colGroups.device.filter(function(item){
								return item.disabled;
							}).map(function(item){
								return item.code;
							});
						
						vm.colForm.device = val?fields:filters;
					},
					cellGroupAllChange(val) {
						var vm = this,
							fields = vm.colGroups.cell.map(function(item){
								return item.code;
							}),
							filters = vm.colGroups.cell.filter(function(item){
								return item.disabled;
							}).map(function(item){
								return item.code;
							});
						
						vm.colForm.cell = val?fields:filters;
					},
					statusGroupAllChange(val) {
						var vm = this,
							fields = vm.colGroups.status.map(function(item){
								return item.code;
							}),
							filters = vm.colGroups.status.filter(function(item){
								return item.disabled;
							}).map(function(item){
								return item.code;
							});
						
						vm.colForm.status = val?fields:filters;
					},
					networkGroupAllChange(val) {
						var vm = this,
							fields = vm.colGroups.network.map(function(item){
								return item.code;
							}),
							filters = vm.colGroups.network.filter(function(item){
								return item.disabled;
							}).map(function(item){
								return item.code;
							});
						
						vm.colForm.network = val?fields:filters;
					},
                    locationGroupAllChange(val) {
						var vm = this,
							fields = vm.colGroups.location.map(function(item){
								return item.code;
							}),
							filters = vm.colGroups.location.filter(function(item){
								return item.disabled;
							}).map(function(item){
								return item.code;
							});
						
						vm.colForm.location = val?fields:filters;
					},

					
					exportcolGroupAllChange(val) {
						var vm = this;

						vm.exportdeviceGroupAllChange(val);
						vm.exportcellGroupAllChange(val);
						vm.exportstatusGroupAllChange(val);
						vm.exportnetworkGroupAllChange(val);
                        vm.exportlocationGroupAllChange(val);
					},
					exportdeviceGroupAllChange(val) {
						var vm = this,
							fields = vm.colGroups.device.map(function(item){
								return item.code;
							}),
							filters = vm.colGroups.device.filter(function(item){
								return item.disabled;
							}).map(function(item){
								return item.code;
							});
						
						vm.exportColForm.device = val?fields:filters;
					},
					exportcellGroupAllChange(val) {
						var vm = this,
							fields = vm.colGroups.cell.map(function(item){
								return item.code;
							}),
							filters = vm.colGroups.cell.filter(function(item){
								return item.disabled;
							}).map(function(item){
								return item.code;
							});
						
						vm.exportColForm.cell = val?fields:filters;
					},
					exportstatusGroupAllChange(val) {
						var vm = this,
							fields = vm.colGroups.status.map(function(item){
								return item.code;
							}),
							filters = vm.colGroups.status.filter(function(item){
								return item.disabled;
							}).map(function(item){
								return item.code;
							});
						
						vm.exportColForm.status = val?fields:filters;
					},
					exportnetworkGroupAllChange(val) {
						var vm = this,
							fields = vm.colGroups.network.map(function(item){
								return item.code;
							}),
							filters = vm.colGroups.network.filter(function(item){
								return item.disabled;
							}).map(function(item){
								return item.code;
							});
						
						vm.exportColForm.network = val?fields:filters;
					},
                    exportlocationGroupAllChange(val) {
						var vm = this,
							fields = vm.colGroups.location.map(function(item){
								return item.code;
							}),
							filters = vm.colGroups.location.filter(function(item){
								return item.disabled;
							}).map(function(item){
								return item.code;
							});
						
						vm.exportColForm.location = val?fields:filters;
					},


					ColumnConfigEn() {
						var vm = this,
							url = '${ctx}/cell/cpeinfos/cellColumnConfig.action',
							sortCol = [];
							
						vm.dragColumns.map(function(item){
							sortCol.push(item.code);
						});

						vm.configShow();

						url = '${ctx}/system/column/setting/insert.action';
						// 保存显示列
						var params = {
								pageName: '6',
								showColumn: vm.showCols.join(','),
								sortColumn: sortCol.join(',')
							};

						axios.post(url, params).then(function(res){
							var innerTb = vm.$refs.monitor.$refs.ctableInner,
								tbStates = innerTb.store.states,
								sortCodes = sortCol,
								storeCols = tbStates.columns;

							tbStates._columns = storeCols.sort(function(n, m) {
								var idxn = sortCodes.indexOf(n.property)==-1?100:sortCodes.indexOf(n.property),
									idxm = sortCodes.indexOf(m.property)==-1?100:sortCodes.indexOf(m.property);

								if( [undefined,'op','connection_status','alarm_count'].includes(n.property) ) {
									idxn = 0;
								}
								if( [undefined,'op','connection_status','alarm_count'].includes(m.property) ) {
									idxm = 0;
								}

								return idxn - idxm;
							});

							innerTb.store.updateColumns();
							$('#gnodeb_ctn').width('99.9%');
							setTimeout(function(){
								$('#gnodeb_ctn').width('100%');
							},2000);
						}).catch(function(){});

						vm.close();
					},
					configShow() {
						var vm = this;

						vm.showProps = vm.showCols;
					},
					close() {
						this.sortGroupShow = false;
					},
					initColumnOrder() {
						var vm = this,
							map = {
								device: vm.colGroups.device.map(function(item){ return item.code;}),
								cell: vm.colGroups.cell.map(function(item){ return item.code;}),
								status: vm.colGroups.status.map(function(item){ return item.code;}),
								network: vm.colGroups.network.map(function(item){ return item.code;}),
                                location: vm.colGroups.location.map(function(item){ return item.code;})
							};

						axios.post('${ctx}/system/column/setting/load/6').then(function(res){
							var data = res.data.data||{},
								sortCodes = data.sortColumn ? data.sortColumn.split(',') : [],
									sortList = vm.dragColumns,
									dragCodes = sortList.map(function(item){ return item.code});

							vm.showProps = data.showColumn ? data.showColumn.split(',') : vm.showCols;
							vm.sortColumns = sortCodes;
								// ------ 
								vm.showProps.map(function(col){
									['device','cell','status','network','location'].map(function(code){
										if(map[code].includes(col) && !vm.colForm[code].includes(col)) vm.colForm[code].push(col);
									});
									// 初始化默认选中项
									if(dragCodes.includes(col) && !vm.dragCol.includes(col)) vm.dragCol.push(col);
								});

							vm.dragColumns = sortList.sort(function(n, m) {
								var idxn = sortCodes.indexOf(n.code)==-1?100:sortCodes.indexOf(n.code),
									idxm = sortCodes.indexOf(m.code)==-1?100:sortCodes.indexOf(m.code);

								return idxn - idxm;
							});

							vm.$nextTick(function(){
								vm.fitOrder();
							})
						});
					},
					fitOrder() {
						var vm = this,
							sortCodes = vm.sortColumns;

						var innerTb = vm.$refs.monitor.$refs.ctableInner,
							tbStates = innerTb.store.states,
								storeCols = tbStates.columns;

							tbStates._columns = storeCols.sort(function(n, m) {
								var idxn = sortCodes.indexOf(n.property)==-1?100:sortCodes.indexOf(n.property),
									idxm = sortCodes.indexOf(m.property)==-1?100:sortCodes.indexOf(m.property);

							if( [undefined,'op','connection_status','alarm_count'].includes(n.property) ) {
									idxn = 0;
								}
							if( [undefined,'op','connection_status','alarm_count'].includes(m.property) ) {
									idxm = 0;
							}

							return idxn - idxm;
						});

						innerTb.store.updateColumns();
						$('#gnodeb_ctn').width('99.9%');
						setTimeout(function(){
							$('#gnodeb_ctn').width('100%');
						},50);
					},
					// 设备表格 选择事件
					selectionChange(s) {
						this.selectedRows = s||[];
					},
					 // 打开已选弹窗
					 openBulkSelectTable(){
						var vm = this;
						vm.bulkSelectShow = true
					},
					// 关闭已选弹窗
					closeBulkSelectTable(){
						var vm = this;
						vm.bulkSelectShow = false;
					},
					// 设备已选表格 清空事件
					clearBulkSelected(){
						var vm = this;
	
						vm.$refs["monitor"].clearSelection();
					},
					// 设备已选表格 单个删除事件
					delBulkSelected(rows){
						var vm = this,
							tabs = 'monitor',
							rowKey = 'small_cell_code';
						vm.selectedRows = vm.selectedRows.filter((items)=>{
							return items[rowKey] != rows[rowKey]
						});
						var selection = this.$refs[tabs].$refs.ctableInner.store.states.selection,
							irow= selection.filter((items)=>{
								return items[rowKey] == rows[rowKey]
							})[0];
						vm.$refs[tabs].toggleRowSelection(irow,false);
						var idx = vm.$refs[tabs].ckList.indexOf(rows[rowKey]);
						vm.$refs[tabs].ckList.splice(idx,1);
					},
                    // 批量同步
                    batchSyncList() {
                        var vm = this;

                        if(vm.selectedRows.length == 0)return
                        vm.syncType = 'batch';
                        vm.syncDialog = true;
                    },
                    // 批量重启
                    batchRebootList(){
                        var vm = this,
                            urls= '${ctx}/task/reboot/batchRebootCell.action',
                            codeList = [],
                            params = {
                                cellCodes:'',
                                isGnb:1
                            };
                        if(vm.selectedRows.length == 0)return
                        vm.selectedRows.map(function(item){
                            codeList.push(item.small_cell_code);
                        });
                        params.cellCodes = codeList.join(',');
                        vm.$confirm('<%=rb.getString("QueDingChongQiSheBei")%>','<%=rb.getString("QueRen")%>',{
							customClass:"warningConfirm",
							confirmButtonText:'<%=rb.getString("QueDing")%>',
							cancalButtonText:'<%=rb.getString("QuXiao")%>',
							type:'warning',
						}).then(()=>{
							axios.post(urls,stringify(params)).then(function(response){
								let data = response.data;
								if ( data.success ){
									vm.$message({
										message: '<%=rb.getString("ChengGong")%>' ,
										type:'success',
									})
									vm.$refs.monitor.refresh();
									vm.$refs.monitor.clearSelection();
								}else {
									vm.$message.error(data.message)
								}
							}).catch(function(error){})
						}).catch(()=>{})
                    },
					// 批量移入回收站设备
					recycleCells(){
						var vm = this,
							params = {},
							urls= '${ctx}/recycle/moveDeviceToRecycle.action?isGnb=1',
							idsList = [];
						if(vm.selectedRows.length <= 0)return
						vm.selectedRows.map((item,index) => {
							idsList.push(item.small_cell_code);
						})
						params.smallCellCodeStr = idsList.join(',');
						var confirmHint ='<div style="font-size:14px;color:#333333">'+ '<%=rb.getString("QueRenJiangSheBeiYiRuHuiShouZhan")%>' +'</div>'+'<div style="font-size:12px;color:#999999">'+ '<%=rb.getString("YiRuHuiShouZhanTiShi")%>' +'</div>';
						vm.$confirm(confirmHint,'<%=rb.getString("QueRen")%>',{
							customClass:"warningConfirm",
							confirmButtonText:'<%=rb.getString("QueDing")%>',
							cancalButtonText:'<%=rb.getString("QuXiao")%>',
							type:'warning',
							dangerouslyUseHTMLString:true
						}).then(()=>{
							axios.post(urls,stringify(params)).then(function(response){
								let data = response.data;
								if ( data.success ){
									vm.$message({
										message: '<%=rb.getString("ChengGong")%>' ,
										type:'success',
									})
									vm.$refs.monitor.refresh();
									vm.$refs.monitor.clearSelection();
								}else {
									vm.$message.error(data.message)
								}
							}).catch(function(error){})
						}).catch(()=>{})
					},
					
					// 判断小区状态 RF状态 显示  1 全部在线 2 部分在线 3 全部不在线
					judgeActiveStatusFat(value){
						var status = '1',
							states = (value+'').split(',');
						if((states.indexOf('0') >= 0  && states.indexOf('1') >= 0) || (states.indexOf('off') >= 0  && states.indexOf('on') >= 0)){
							status = '2'
						}else if((states.indexOf('0') >= 0  && states.indexOf('1') < 0) || (states.indexOf('off') >= 0  && states.indexOf('on') < 0)){
							status = '3'
						}else if((states.indexOf('0') < 0  && states.indexOf('1') >= 0) || (states.indexOf('off') < 0  && states.indexOf('on') >= 0)){
							status = '1'
						}
						return status
					},
					// 判断小区装填 RF状态  在线数 与 总数
					judgeActiveNumOrAllNumFat(value,type){
						var activeNum = [],
							allNum = [],
							states = (value+'').split(','),
							val = 0;
						states.map((item,index)=>{
							if(item == '1' || item == 'on'){
								activeNum.push(item)
							}
							allNum.push(item)
						})

						if(type == 'active'){
							val = activeNum.length
						}else{
							val = allNum.length
						}
						return val
					},
					// 生成 小区状态 RF状态 集合
					parseCellAndRfStatus(value){
						var states = (value+'').split(',');
						return states
					},
                    parseParamsOnNum(str){
                        var onNum = [],
                            list = [];
                        if(str) {
                            list = eval('('+str+')');
                        }else {
                            list = [];
                        }
                        list.map((item,index)=>{
                            if(item.Status == '1'){
                                onNum.push(item)
                            }
                        })
                        return onNum.length
                    },
                    parseParams(str) {
                        if(str) {
                            return eval('('+str+')');
                        }else {
                            return [];
                        }
                    },
                    amfStatusFmt(str){
                        var amfList = eval('('+str+')');
                        var amfStatus,textVal,value,hasDisconn=false,hasConn=false;
                        amfList.map(function(item){
                            if (item.Status == '1'){
                                hasConn = true;
                            }else if(item.Status == '0' || item.Status == null || item.Status == 'null'){
                                hasDisconn = true
                            }
                        })
                        
                        if (hasDisconn == true){
                            if (hasConn == false) {
                                amfStatus ='offStatusCls';
                                textVal = '<%= rb.getString("MMEWeiLianJie")%>'
                            }else{
                                amfStatus ='offOrOnStatusCls';
                                textVal = '<%= rb.getString("MMEYiLianJie")%>'
                            }
                        }else {
                            amfStatus ='';
                            textVal = '<%= rb.getString("MMEYiLianJie")%>'
                        }
                        
                        value = "<span class='"+ amfStatus +"'>" + textVal +"</span>"
                        return value;
                    },
					// 获取自定义label信息
					getCustomLabelData() {
						var vm = this;

						axios.post('${ctx}/cell/columnAlias/queryColumnAliasConfigs.action').then(function(response){
							var data = response.data || [];

							data.forEach(function(item){
								if(item.columnName == 'remark'){
									vm.currentRemarkLabel = item.columnAlias || 'Remark';
									vm.remarkLabelInput = vm.currentRemarkLabel;
								}
							});
						}).catch(function(error){});
					},
					// Remark label 编辑方法
					startEditRemarkLabel() {
						this.editingRemarkLabel = true;
						this.remarkLabelInput = this.currentRemarkLabel;
					},
					saveRemarkLabel() {
						var vm = this,
							params = {
								columnName: 'remark',
								columnAlias: this.remarkLabelInput.trim()
							};
						if(this.remarkLabelInput.trim() === ''){
							vm.$message.warning('<%=rb.getString("QingShuRuBiTianXiang")%>');
							return;
						}
						let paramsData = JSON.stringify(params);
						axios.post('${ctx}/cell/columnAlias/setColumnAlias.action',paramsData,{headers:{'Content-Type':'application/json;charset=utf-8'}}).then(function(response){
							var data = response.data;

							if(data.success){
								vm.currentRemarkLabel = vm.remarkLabelInput.trim();
								vm.editingRemarkLabel = false;
								vm.syncRemarkLabel();
								vm.$message.success('<%=rb.getString("ChengGong")%>');
							}else{
								vm.$message.error(data.message);
							}
						}).catch(function(error){});
					},
					cancelEditRemarkLabel() {
						this.editingRemarkLabel = false;
						this.remarkLabelInput = this.currentRemarkLabel;
					},
					// 同步其他页面的Remark label
					syncRemarkLabel() {
						var vm = this,
							newLabel = vm.currentRemarkLabel,
							vueInstances = [
								{name: 'enbvm', instance: typeof enbvm !== 'undefined' ? enbvm : null},
								{name: 'gsmvm', instance: typeof gsmvm !== 'undefined' ? gsmvm : null},
								{name: 'egwRegisterVue', instance: typeof egwRegisterVue !== 'undefined' ? egwRegisterVue : null},
								{name: 'gnbOverviewVue', instance: typeof gnbOverviewVue !== 'undefined' ? gnbOverviewVue : null},
                                {name: 'enbDetailVue', instance: typeof enbDetailVue !== 'undefined' ? enbDetailVue : null},
                                {name: 'gsmSettingOverviewVue', instance: typeof gsmSettingOverviewVue !== 'undefined' ? gsmSettingOverviewVue : null},
							];

						vueInstances.forEach(function(vue){
							if(vue.instance && vue.instance.currentRemarkLabel !== undefined){
								vue.instance.currentRemarkLabel = newLabel;
							}
							if(vue.instance && vue.instance.remarkLabelInput !== undefined){
								vue.instance.remarkLabelInput = newLabel;
							}
						});
					},
                    synchronizeGPS(code) {
                        var vm = this;

                        $.ajax({
                            url: '${ctx}/cell/topo/syncGPSInfo.action',
                            type: 'post',
                            data: {cell_code: code},
                            dataType: 'json',
                            success: function(data) {
                                if(data.success){
                                    vm.$refs.monitor.refresh();
                                }else{
                                    showMsg('error_msg',data["message"])
                                }
                            }
                        });
                    },
				},
				created() {
					var vm = this;

					// 初始化 colGroups.device 为计算属性 colGpDevice 的值
					vm.colGroups.device = vm.colGpDevice;
					
					vm.getCustomLabelData();

					//vm.initColumnOrder();
				},
				mounted() {
					var vm = this;

					eventBus.$off('sync-setting').$on('sync-setting',this.goSetting);
					eventBus.$off('close-gnbMonitorsettingSlide').$on('close-gnbMonitorsettingSlide',this.hideSlide);
					
					this.init();
					vm.initColumnOrder();
					closeLoading();

					this.queryLatestCollectInfo();
				}
			})
			/**
			* 告警级别格式化
			* @param value{string}: 告警数
			* @param rowData{object}: 列表行数据
			* @param rowIndex{string}: 列表行数据对应下标
			**/
			function alarmFormatter(value,rowDatas,index){
				var alarmCount = rowDatas.alarm_count;
				var alarmServerity = rowDatas.alarm_serverity;
				var smallCellCode = rowDatas.small_cell_code;
				// 告警级别转义
				if(alarmServerity == "31001"){
					value = "<span class='alarmCritical alarmListSty'>"+alarmCount+"</span>"
				}else if(alarmServerity == "31002"){
					value = "<span class='alarmMajor alarmListSty'>"+alarmCount+"</span>"
				}
				else if(alarmServerity == "31003"){
					value = "<span class='alarmMinor alarmListSty'>"+alarmCount+"</span>"
				}else if(alarmServerity == "31004"){
					value = "<span class='alarmWarning alarmListSty'>"+alarmCount+"</span>"
				}else{
					value = "0"
				}

				return value;
			}
			/**
			* 小站激活状态-内容处理
			* @param value{string}: 当前状态值
			* @param rowData{object}: 列表行数据
			* @param rowIndex{string}: 列表行数
			**/
			function cellStateFormatter(value, rowData, rowIndex){
				if (value == null) {
					return null;
				}
				else if (value == "1") {
					val = '<%= rb.getString("JiHuo")%>';
					value = "<div class=''>"+(val)+"</div>"
				}
				else if (value == "0") {
					//状态不一样展示的文字也不一样
					val = '<%= rb.getString("QuJiHuo")%>';
					value = "<div class='inactiveStatusItem'>"+(val)+"</div>"
				}
				return value;
			}
		</script>
	</body>
</html>