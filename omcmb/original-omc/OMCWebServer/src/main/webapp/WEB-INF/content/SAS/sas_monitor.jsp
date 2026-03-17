<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page import="com.baicells.omc.busi.system.login.entity.UserInfo" %>
<%@page import="com.baicells.omc.busi.utils.ComConstants" %>
<%
	UserInfo user = (UserInfo) session.getAttribute(ComConstants.SESSION_KEY);
%>

<script type="text/javascript" src="${ctx}/js/element/Sortable.min.js?_=${omc_ver}"></script>
<script type="text/javascript" src="${ctx}/js/element/vuedraggable.umd.min.js?_=${omc_ver}"></script>

<style>
	#sasMonitorPage{
		overflow: hidden;
		min-width: 1200px;
		width: 100%;
		height: 100%;
		background: #FFFFFF;
		position: relative;
	}
	.el-notification__group .el-icon-close{
		position: absolute !important
	}
	.el-message__closeBtn{
		position: absolute !important;
		right:15px !important
	}
	.slide-content{
		padding:0px !important;
	}
	.el-pagination .el-select .el-input{
		width:85px;
	}
	.el-pagination .el-select .el-input .el-input__inner{
		width:85px;
		background:#fff !important;
	}
	.msgNotify{
		word-wrap: break-word;
		word-break: break-all
	}
	.msgNotify .el-notification__title{
		margin-left: 5px
	}
	.settingBox{
		background: rgba(0,0,0,0.3);
	}
	.settingBoxForm .el-form-item__label{
		line-height: 24px
	}
	.settingBoxForm .el-input__inner{
		height: 26px !important;
	}
	.setEnb{
		margin:30px
	}
	.setEnb .el-form-item__label{
		width: 150px
	}
	.pageFooter{
		height:55px;
		border-top:1px solid #EEEEEE;
		padding:15px 0 15px 40px;
		box-sizing:border-box;
		position: absolute;
		bottom:0px;
		width:100%;
	}
	.pageFooter .el-button, .footerBtn{
		width:86px;
		height:25px;
		border-radius:unset;
		padding:0px;
	}
	.tslide {
		left:unset;
		right:0px;
		top:45px;
		box-shadow:0 0 50px rgba(158,200,222,0.35);
	}
	.noDateContainer{
		width:100%;
		height:100%;
		display:flex;
		justify-content:center;
		align-items:center;
	}
	.noDatePage{
		font-weight:700;
		font-style:normal;
		font-size:18px;
		color:#666666;
	}
	.f12{
		font-size: 12px
	}
	.chengong .el-icon-status-active:before{
		color:#67D972;
		font-size: 20px
	}
	.shibai .el-icon-status-active:before{
		color:#E88282;
		font-size: 20px
	}
	.el-icon-status-conn-off:before ,.el-icon-status-disable:before , .el-icon-status-enable:before{
		font-size: 20px
	}
	.ml5{margin-left:5px}
	.el-icon-circle-warning:before{
		font-size: 20px
	}
	
	.el-icon-status-conn-on:berfoe{
		font-size: 20px
	}
	.userBox .el-form-item__error{
		margin-left:70px
	}

	.exportDrop{
		width:30px;
		height:30px;
		position:absolute;
		right:20px;
		top:47px;
	}
	.sasStatistics{
		position:relative;
		height:26px;
		z-index:99;
		display:flex;
	}
	.sasStatistics div{
		box-sizing: border-box;
	}
	.sasStatistics .grantSuspended{
		margin:0 15px;
	}
	.sasStatistics .enableStatistics{
		display:flex;
		border:1px solid #DCDFE6;
		height:26px;
		line-height:26px;
		border-radius:4px;
		margin-right:15px;
	}
	.sasStatistics .enableStatistics .enableTitleStatistics{
		border-right:1px solid #DCDFE6;		
		color:#666666;
	}
	.sasStatistics .enableStatistics .enableNumStatistics{
		color:#333333;
		font-weight:bold;
	}
	.sasStatistics .enableStatistics .titleCommon{
		padding:0 15px;
		font-size:12px;
	}
	.sasStatistics .enableStatistics .enableBg{
		background:#DFE9FF;
	}
	.sasStatistics .enableStatistics .grantSuspendedBg{
		background:#F9E6E6;
	}
	.sasStatistics .enableStatistics .authorizedBg{
		background:#EBF4E8;
	}
	.sasStatedBox .el-icon::before{
		font-size: 16px;
		color: #F2B354;
	}
	.virtualIconClass{
		display: inline-block;
		position: relative;
		color: #4D84FF;
		border: 1px solid #4D84FF;
		top:0px;
		left: 2px;
		background-color: #EDF6FF;
		height: 18px;
		width: 50px;
		border-radius: 5px;
		line-height: 17px;
		text-align: center;
		font-size: 12px;
		-webkit-transform: scale(0.84,0.84);
	}
	.sasEnableClickBox{
		height: 20px;
		width: 50px;
		position: absolute;
		top:0px;
		left: 0px;
		right: 0px;
		bottom: 0px;
		margin: auto;
		z-index: 66;
		cursor: pointer;
		opacity: 0;
	}
	#sasMonitorPage .sasMonitorOptCls >div{
		margin-right: 3px;
	}

	#sasMonitorPage .showHideItem {
		position:absolute;
		background:white;
		z-index:888;
		padding-top: 10px;
		padding-left: 10px;
		width: 600px;
		top: 65px;
		left: 0px;
		display: none;
		box-shadow:5px 10px 23px 0px rgba(0,0,0,0.15);
	}
	#sasMonitorPage .showHideItem input{
		margin-top:-2px;
		margin-bottom:1px;
		vertical-align:middle;
		margin-right:20px;
	}
	#sasMonitorPage .showHideItem .select-all-cls {
		padding: 10px 0 0 9px;
		display: flex;
		align-items: center;
	}
	#sasMonitorPage .select-all-cls > i {
		margin-right: 5px;
	}
	#sasMonitorPage .showHideItem .select-all-cls > span {
		font-size: 14px;
		font-weight: bold;
		margin-left: 10px;
	}
	#sasMonitorPage  .showHideItem .el-icon-close1::before {
		color: #333;
	}

	#sasMonitorPage .list-group > span {
		display: flex;
		flex-direction: column;
		flex-wrap: wrap;
		padding-left: 15px; 
		height: 230px;
	}
	#sasMonitorPage .list-group-item {
		display: inline-block;
		position: relative;
		padding: 0px 10px;
		margin: 3px 5px;
		width: 215px;
		border: 0px dashed #ddd;
		cursor: move;
	}
	#sasMonitorPage .grantsIcon::before{
		font-size: 15px;
	}
	.grantsPopoverClass .grantsItemBoxCls{
		padding:20px;
		border-top:1px solid #E9E9E9;
	}
	.grantsPopoverClass .grantsItemBoxCls .grantsItemCls{
		padding:5px 0px;
	}
	.grantsPopoverClass .grantsItemCls > div:first-child{
		width: 120px;
		white-space:nowrap;
		color:#333333;
	}
	.grantsPopoverClass .grantsItemCls > div:last-child{
		white-space:pre-wrap;
		padding-left: 5px;
		max-height: 140px;
		overflow: auto;
	}
	#sasMonitorPage .monitorContentBoxCls{
		height: 100%;
		position: relative;
		display: flex;
	}
	#sasMonitorPage .monitorContentBoxCls >div:first-child{
		flex: 1;
		position: relative;
	}
	#sasMonitorPage .monitorContentBoxCls >div{
		background-color: #FFFFFF;
		box-shadow: 0px 0px 10px 1px #E9EDF9;
		border-radius: 10px;
		border: 1px solid #E9EDF9;
		overflow: hidden;
	}
	#sasMonitorPage .monitorContentBoxCls .importSasCertBoxCls{
		flex: 0 1 360px;
		margin-left: 10px;
		position: relative;
	}
	#sasMonitorPage .rightOutBoxHeadCls{
		height: 50px;
		display: flex;
		align-items: center;
		font-weight: 600;
		font-size: 14px;
		justify-content: space-between;
		padding: 0px 20px;
		border-bottom: 1px solid #E9EDF9;
	}
	#sasMonitorPage .rightItemMainBox{
		padding: 20px;
	}
	#sasMonitorPage .rightItemMainBox .el-form-item{
		margin-bottom: 20px;
	}
	.greyIcon::before{
		color: #7A7992;
		font-size: 14px;
	}
	.whirtIcon::before{
		color: #FFFFFF;
		font-size: 14px;
	}
	#sasMonitorPage .rightItemMainBox .el-input-group__append{
		background-color: #fff;
		padding: 0 10px;
	}
	#sasMonitorPage .labelSlotCls > span{
		color: #999999;
		font-size: 12px;
		margin-left: 10px;
	}
	#sasMonitorPage .importSasCertBoxCls .footer{
		width:100%;
		border-top:1px solid #E9E9E9;
		position:absolute;
		bottom:1px;
		height:50px;
		background:#FFFFFF;
		z-index:99;
		display: flex;
		align-items: center;
		border-radius: 0px 0px 10px 10px;
	}
	.w300{width: 300px;}
	.tooltipCls.is-dark{
		background : #959595 ;
		color : #FFFFFF ;
	}
	.tooltipCls[x-placement^=top] .popper__arrow ,
	.tooltipCls[x-placement^=top] .popper__arrow::after{
		border-top-color: #959595!important;
	}

	.tooltipCls[x-placement^=bottom] .popper__arrow ,
	.tooltipCls[x-placement^=bottom] .popper__arrow::after {
		border-bottom-color: #959595!important;
	}
	.tooltipCls[x-placement^=right] .popper__arrow ,
	.tooltipCls[x-placement^=right] .popper__arrow::after {
		border-right-color: #959595!important;
	}
	.tooltipCls[x-placement^=left] .popper__arrow ,
	.tooltipCls[x-placement^=left] .popper__arrow::after {
		border-left-color: #959595!important;
	}
	#sasMonitorPage .successIcon{
		background: rgba(103, 217, 114, 0.2)!important;
	}
	#sasMonitorPage .successIcon .el-icon::before{
		color:#4ED76E;
	}
	.allCellPopoverClass .el-popover__title{
		padding: 5px;
		margin-bottom: 0px;
	}
	.allCellListCls {
		position: relative;
		display: flex;
		max-width: 800px;
		flex-wrap: wrap;
		max-height: 500px;
		overflow: auto;
	}
	.allCellListCls-item {
		padding: 10px 20px;
		margin: 0px 5px 10px 5px;
		border: 1px solid #E9E9E9;
	}
	#sasMonitorPage .cellItemBoxCls{
		display: flex;
		align-items: center;
	}
	#sasMonitorPage .cellItemBoxCls .cellItemOmitCls{
		white-space: nowrap;
		display:inline-block;
		overflow: hidden;
		text-overflow: ellipsis;
		max-width: 90%;
	}
	#sasMonitorPage .el-ctable-toolbar{
		padding: 0px 0px 10px 0px!important;
	}
	#sasMonitorPage .el-ctable th>.cell {
		max-height: 25px !important;
	}
</style>
<div class="overflow-cls">
<!-- SAS 监控功能主页面 -->
<div id="sasMonitorPage">
	<!-- 没有开启SAS功能的用户展示的提示信息 -->
	<div v-show='isNoData' class="noDateContainer" >
        <span class="noDatePage"><%=rb.getString("DingYueCBRSFuWu") %></span>
    </div> 
	
	<!-- 开启SAS功能的用户可查看的监控页面 -->
	<div class="monitorContentBoxCls">
		<div v-show='listShow && !isNoData' id="monitor_ctner">   
			<div v-show="optBtnShow" :class="sasCertStatus ? 'newIconBoxCls-bt successIcon' : 'newIconBoxCls-bt'" style="right:164px;top:42px;" @click="sasCertImport" :tip="sasCertStatus ? '<%=rb.getString("ZhengShuYiQianFa") %>' : '<%=rb.getString("ZhengShuWeiQianFa") %>'">
				<span class="el-icon el-icon-circle-CPISuccess"></span>
			</div>
			<div v-show="optBtnShow" class="newIconBoxCls-bt" style="right:128px;top:42px;" @click="sasImportClick" tip='<%=rb.getString("DaoRu") %>'>
				<span class="el-icon el-icon-operation-import"></span>
			</div>
			<div v-show="optBtnShow" class="newIconBoxCls-bt" style="right:92px;top:42px;" @click="sasSetting" tip='<%=rb.getString("SheZhi") %>'>
				<span class="el-icon el-icon-operation-settings"></span>
			</div>
			<div class="newIconBoxCls-bt" style="right:56px;top:42px;" @click="sasMainLog" tip='<%=rb.getString("RiZhi") %>'>
				<span class="el-icon el-icon-operation-details"></span>
			</div>
			<el-dropdown @command="exportCSV" trigger="click" class="exportDrop">
				<div class="newIconBoxCls-bt" style="right:0px;top:-5px;" tip='<%=rb.getString("DaoChu") %>'>
					<span class="el-icon el-icon-operation-export"></span>
				</div>
				<el-dropdown-menu slot="dropdown">
					<el-dropdown-item command="">Export for monitor</el-dropdown-item>
					<el-dropdown-item command="google">Export to Google SAS</el-dropdown-item>
					<el-dropdown-item command="templateData">Export for CBSD Params</el-dropdown-item>
				</el-dropdown-menu>
			</el-dropdown>
			
			<!-- 主页面区域 -- tab页 -->
			<el-tabs v-model="activeName" class="newTabs" @tab-click='tabClick' style="height:100%;">
				<!--CBSD eNB -->
				<el-tab-pane v-if="writableMap['CODE_ENB_MONITOR'] != undefined && activeName == 'eNB'" label='<%=rb.getString("PeiZhiGuanLi")%>' name="eNB"> <!--:url="enbUrl" :data="enbTableData"  -->
					<el-ctable id="enbSASMonitorTable" ref="enbTable" 
						:url="enbUrl"  
						:limit="limitBatch"
						@sort-change="sortChangeEnb"
						:row-key="'small_cell_code'" 
						:query-params="params_sasEnb" 
						@selection-change='batchSelect' 
						@load-success="tableLoadSuccess" >
						<!-- 高级查询 -- eNb-->
						<template slot="toolbar">
							<div class="toolbarHeadBtnBoxCls">
								<div v-show="optBtnShow" class="selectBlukBoxCls">
									<div class="selectMain">
										 <div class="bulkSelectBtnBoxCls"  @click="openBulkSelectTable">
                                            <span class="el-icon-selected el-icon"></span>
                                            <span class="bulkSelectNumBoxCls">( {{selectionData.length}} )</span>
                                        </div>
										<div class="selectTableBoxCls" style="position: absolute;top: 38px;left: 0px;" v-show="enbBulkSelectShow">
											<div class="selectBoxTitle">
												<span><%=rb.getString("YiXuan")%></span>
												<span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="closeBulkSelectTable()"></span>
											</div>
											<div class="selectBoxMain">
												<div class="tableInfoCls">
													<div class="tableInfoHeader">
														<div><%=rb.getString("YiXuanSheBei")%></div>
														<div @click="clearBulkSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span>Clear</div>
													</div>
													<el-ctable 
														id="bulkSelectTable" 
														ref="bulkSelectTable" 
														:data="selectionData" 
														:showHeader="false"
														:rownumber="false"
														:front-pagination="true"
														:row-key="'small_cell_code'"
														height="270px" pagination="true" >
														<el-table-column prop="serialNumber" v-if="false"></el-table-column>
														<el-table-column width="588">
															<template slot-scope="scope" >
																<div class="tableItemCls">
																	<span>{{scope.row.serialNumber}}</span>
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
								<div v-show="optBtnShow" :class="selectionData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="setInstall">
									<span class="el-icon el-icon-operation-settings"></span>
									<span><%=rb.getString("SheZhi")%></span>
								</div>
								<div v-show="optBtnShow" :class="selectionData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="batchCbrsas('on')">
									<span class="el-icon el-icon-status-enable"></span>
									<span>SAS ON</span>
								</div>
								<div v-show="optBtnShow" :class="selectionData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="batchCbrsas('off')">
									<span class="el-icon el-icon-status-disable"></span>
									<span>SAS OFF</span>
								</div>
							</div>
							<div style="position:relative;">
								<div id="tableHeadQuery" class="tableHeadQueryBoxCls">
									<div class="headQueryBox">
										<div class="queryGroup">
											<el-input v-model="enb_search_text" @keyup.enter.native="query" @focus="queryInputFocus" @blur="queryInputBlur" :placeholder='placeholderText' style="width:260px;"></el-input>
											<i @click='query' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
										</div>
									</div>
									<el-popfilter style="margin: 0 5px;"
										type="single"
										label='<%=rb.getString("ShiFouJiHuo") %>'
										v-model="queryForm.opStatus"
										:list="opStatusList"
										@check-change="advanceQuery">
									</el-popfilter>
									<el-popfilter style="margin: 0 5px;"
										type="single"
										label='<%=rb.getString("ShiJiLeiXing") %>'
										v-model="queryForm.cbsdType"
										:list="cbsdTypeList"
										@check-change="advanceQuery">
									</el-popfilter>
									<div class="advancedQueryItemBox"  style="background: #FFF;margin-right:10px;" @click="clearFilterClick">
										<%=rb.getString("QingKongShaiXuan")%>
									</div>
									<!-- SAS 数据统计-->
									<div class="sasStatistics">
										<div class="enableStatistics">		
											<span class="enableTitleStatistics titleCommon enableBg"><%=rb.getString("SASKaiGuan") %></span>
											<span class="enableNumStatistics titleCommon">{{sasEnableCount}}/{{totalCount}}</span>
										</div>
										<div class="enableStatistics">		
											<span class="enableTitleStatistics titleCommon authorizedBg">Authorized</span>
											<span class="enableNumStatistics titleCommon">{{authorizedCount}}/{{totalCount}}</span>
										</div>
										<div class="enableStatistics">		
											<span class="enableTitleStatistics titleCommon grantSuspendedBg">Grant-Suspended</span>
											<span class="enableNumStatistics titleCommon">{{grantSuspendedCount}}/{{totalCount}}</span>
										</div>
									</div>
								</div>
								<span class="el-icon-operation-settings el-icon" 
								style="position:absolute;left:15px;bottom:-60px;z-index:99;width:30px;height: 40px;box-shadow:none;" 
								@click="columnSetting"></span>
							</div>
							<div class="showHideItem" id="enb_sas_column_setting">
								<div class="select-all-cls" style="padding-left: 10px;">
									<el-checkbox :indeterminate="!dragAll" v-model="dragAll" @change="dragAllChange"></el-checkbox> 
									<span><%=rb.getString("QuanXuan")%></span>
								</div>
								<el-checkbox-group v-model="dragCol">
									<draggable
										class="list-group"
										v-model="sortColumns"
										v-bind="dragOptions">
										<transition-group type="transition" :name="!drag? 'flip-list':null">
											<div v-for="(col,idx) in sortColumns" :key="col.field" class="list-group-item">
												<el-checkbox :label="col.field" :key="col.field" :disabled="col.disabled">{{col.label}} </el-checkbox>
												<span v-if="false" style="position: absolute;right: 5px;top: 5px;">{{idx+1}}</span>
											</div>
										</transition-group>
									</draggable>
								</el-checkbox-group>

								<div class="windowButtonGroup" style="float:none !important;margin:20px 0 20px 30px">			
									<a class="linkbutton linkbutton_trend" @click="enbColumnConfig"><span><%=rb.getString("QueDing")%></span></a>
									<a class="linkbutton linkbutton_nowanna" @click="enbCancelConfig"><span><%=rb.getString("QuXiao")%></span></a>
								</div>
							</div>
						</template>
						
						<el-table-column label='' width="50" type="selection" :reserve-selection="true" :selectable = "isDisabled" v-if="optBtnShow"></el-table-column>
						<!-- 主列表 -->
						<el-table-column label='' width="90" prop="">
							<template slot-scope="scope">
								<!-- 如果是辅助小区，增加 carrierType字段，carrierType字段值为'scell'，表格操作项及SAS开关禁止点击 -->
								<div v-if="scope.row.carrierType == 'scell'" class="sasMonitorOptCls">
									<div class="el-icon el-icon-operation-settings disabled" title="<%=rb.getString("SheZhi") %>"></div>
									<div v-if="scope.row.cbsdType =='virtual'&&optBtnShow" class="el-icon el-icon-operation-delete disabled" title="<%=rb.getString("ShanChu") %>"></div>
									<div class="el-icon el-icon-operation-details" title="<%=rb.getString("RiZhi") %>" @click="sasLogs(scope.row)"></div>
								</div>
								<div v-else class="sasMonitorOptCls">
									<div class="el-icon el-icon-operation-settings" title="<%=rb.getString("SheZhi") %>" @click="goProperties(scope.row)"></div>
									<div v-if="scope.row.cbsdType =='virtual'&&optBtnShow" class="el-icon el-icon-operation-delete" title="<%=rb.getString("ShanChu") %>" @click="deleteVirtual(scope.row)"></div>
									<div class="el-icon el-icon-operation-details" title="<%=rb.getString("RiZhi") %>" @click="sasLogs(scope.row)"></div>
								</div>
							</template>
						</el-table-column>
						
						<el-table-column label='<%=rb.getString("SASKaiGuan") %>' width="110" prop="sasEnable" sortable header-align="left" align="center">
							<template slot-scope="scope" v-if="scope.row">
								<!-- 如果是辅助小区，增加 carrierType字段，carrierType字段值为'scell'，表格操作项及SAS开关禁止点击 -->
								<template v-if="!optBtnShow">
									<div v-if="scope.row.sasEnable == 'on' || scope.row.sasEnable == 'off'">
										<el-switch v-model="scope.row.sasEnable" style="height: 18px;margin-left: 10px;" :disabled="true"
											active-color="#4D84FF"
											active-value="on"
											inactive-value="off">									
										</el-switch>
									</div>
									<div v-else> -- </div>
								</template>
								<template v-else>
									<!-- 根据在线不在线状态判断显示 SAS 开关的状态 -->					
									<div v-if="scope.row.connection_status != 'Off'">
										<div v-if="scope.row.sasEnable == 'on' || scope.row.sasEnable == 'off'" style="display:flex;">
											<div v-if="scope.row.carrierType == 'scell'">
												<el-switch :ref="scope.row.serialNumber" v-model="scope.row.sasEnable" style="height: 18px;margin-left: 10px;" :disabled="true"
													active-color="#4D84FF"
													active-value="on"
													inactive-value="off">
												</el-switch>
											</div>
											<div v-else style="position:relative;">
												<el-switch :ref="scope.row.serialNumber" v-model="scope.row.sasEnable" style="height: 18px;margin-left: 10px;"
													active-color="#4D84FF"
													active-value="on"
													inactive-value="off">
												</el-switch>
												<div class="sasEnableClickBox" @click="sasEnabelChange(scope.row)"></div>
											</div>
											<div v-if="scope.row.sasEnable == 'off' && scope.row.scheduleTime">
												<el-popover style='word-wrap:break-word' placement='bottom' trigger='hover' width='260'>
													<div style="padding:20px;width:200px;height:60px;position:relative;">
														<div style="margin-bottom:10px;">
															<div><%= rb.getString("ZhiXingShiJian")%>:</div>
															<div style="padding-left:20px;">{{scope.row.scheduleTime}}</div>
														</div>
													</div>
													<span slot='reference' style="margin-left:10px;" class="el-icon el-icon-operation-time"></span>
												</el-popover>
											</div>
										</div>
										<div v-else> --</div>
									</div>
									
									<div v-else-if="scope.row.connection_status == 'Off'" style="display:flex;">
										<div v-if="scope.row.sasEnable == 'on' || scope.row.sasEnable == 'off'">
											<div v-if="scope.row.sasEnable == 'on'"  style="position:relative;">
												<el-switch :ref="scope.row.serialNumber" v-model="scope.row.sasEnable" style="height: 18px;margin-left: 10px;"
													:disabled="scope.row.cbsdType =='virtual'"
													active-color="#4D84FF"
													active-value="on"
													inactive-value="off">
												</el-switch>
												<div v-if="scope.row.cbsdType == 'virtual'" class="disabled"></div>
												<div v-if="scope.row.cbsdType != 'virtual'" class="sasEnableClickBox" @click="sasEnabelChange(scope.row)"></div>
											</div>
											<div v-if="scope.row.sasEnable == 'off'">
												<el-switch v-model="scope.row.sasEnable" style="height: 18px;margin-left: 10px;" :disabled="true"
													active-color="#4D84FF"
													active-value="on"
													inactive-value="off">									
												</el-switch>
											</div>
										</div>
										<div v-else> -- </div>
										<div v-if="scope.row.sasEnable == 'off' && scope.row.scheduleTime">
											<el-popover style='word-wrap:break-word' placement='bottom' trigger='hover' width='260'>
												<div style="padding:20px;width:200px;height:50px;position:relative;">
													<div style="margin-bottom:20px;">
														<div><%= rb.getString("ZhiXingShiJian")%>:</div>
														<div style="padding-left:20px;">{{scope.row.scheduleTime}}</div>
													</div>
												</div>
												<span slot='reference' style="margin-left:10px;" class="el-icon el-icon-operation-time"></span>
											</el-popover>
										</div>								
									</div>
									
									<div v-else></div>
								</template>
							</template>
						</el-table-column>
									
						<el-table-column prop="connection_status" sortable min-width="50" width="50" >
							<template slot-scope="scope">
								<div :class="{
									'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
									'':scope.row.have_connected==2,
									'conn_exc':scope.row.connection_status=='Exception',
									'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:20px;'></div>
							</template>
						</el-table-column>

						<el-table-column v-for="col in columns" :prop="col.field" :label="col.label" :sortable="col.sortable" :min-width="col.width" v-if="showColums.includes(col.field)">
							<template slot-scope="scope">
								<template v-if="col.field == 'serialNumber'">
									{{scope.row.serialNumber}}<div class="virtualIconClass" v-if="scope.row.cbsdType =='virtual'">Virtual</div>
								</template>
								<template v-if="col.field == 'directMode'">
									<div v-if="scope.row.directMode =='on'"><%=rb.getString("ZhiLianMoShi") %></div>
									<div v-if="scope.row.directMode =='off'"><%=rb.getString("ShuJuChuLiMoShi") %></div>
								</template>
								<template v-if="col.field == 'hostName'">
									<!--<div v-if="scope.row.cellType == 'TC'">
										<div  v-if="!['','NULL','null',null,undefined].includes(scope.row.hostName)" class="cellItemBoxCls">
											<span>{{scope.row.hostName[0]}}</span>
											<el-popover title="All" popper-class="allCellPopoverClass">
												<span style="color:#4d84ff;" slot="reference">
													[ <span v-html="scope.row.hostName.length"></span> 
													<i class="el-icon el-icon-common-arrow-down" style="font-size: 12px; zoom: 0.7;"></i>]
												</span>
												<div class="allCellListCls">
													<i class="el-icon el-icon-close" style="position: absolute; top: -30px; right: 0px;" onclick="document.body.click()"></i>
													<div class="allCellListCls-item" v-for="(item,index) in scope.row.hostName">
														<div>Cell{{index+1}}:</div>
														<div>{{item}}</div>
													</div>
												</div>
											</el-popover>
										</div>
									</div>
									<div v-if="scope.row.cellType !== 'TC'">
										{{scope.row.hostName}}
									</div>-->
									{{scope.row.hostName}}
								</template>
								<template v-if="col.field == 'state'">
									<div v-if="scope.row.cellType == 'TC'">
										<div  v-if="!['','NULL','null',null,undefined].includes(scope.row.state)" class="cellItemBoxCls">
											<span v-html="sasStateFmt(scope.row.state[0])"></span>
											<el-popover title="All" popper-class="allCellPopoverClass">
												<span style="color:#4d84ff;" slot="reference">
													[ <span v-html="scope.row.state.length"></span> 
													<i class="el-icon el-icon-common-arrow-down" style="font-size: 12px; zoom: 0.7;"></i>]
												</span>
												<div class="allCellListCls">
													<i class="el-icon el-icon-close" style="position: absolute; top: -30px; right: 0px;" onclick="document.body.click()"></i>
													<div class="allCellListCls-item" v-for="(item,index) in scope.row.state">
														<div>Cell{{index+1}}:</div>
														<div v-html="sasStateFmt(item)">{{item}}</div>
													</div>
												</div>
											</el-popover>
										</div>
									</div>
									<div v-if="scope.row.cellType !== 'TC'">
										<span v-html="sasStateFmt(scope.row.state)"></span>
									</div>
								</template>
								<template v-if="col.field == 'cpiState'">
									<div v-if="scope.row.cellType == 'TC'">
										<div v-if="!['','NULL','null',null,undefined].includes(scope.row.cpiState)" class="cellItemBoxCls">
											<span style="display:inline-block;">{{scope.row.cpiState[0]}}</span>
											<el-popover title="All" popper-class="allCellPopoverClass">
												<span style="color:#4d84ff;" slot="reference">
													[ <span v-html="scope.row.cpiState.length"></span> 
													<i class="el-icon el-icon-common-arrow-down" style="font-size: 12px; zoom: 0.7;"></i>]
												</span>
												<div class="allCellListCls">
													<i class="el-icon el-icon-close" style="position: absolute; top: -30px; right: 0px;" onclick="document.body.click()"></i>
													<div class="allCellListCls-item" v-for="(item,index) in scope.row.cpiState">
														<div>Cell{{index+1}}:</div>
														<div>{{item}}</div>
													</div>
												</div>
											</el-popover>
										</div>
									</div>
									<div v-if="scope.row.cellType !== 'TC'">
										{{scope.row.cpiState}}
									</div>
								</template>
								<template v-if="col.field == 'opStatus'">
									<div v-if="scope.row.opStatus == '1'" class="chengong">
										<span class="el-icon el-icon-status-active">
											<span class="f12 ml5"><%= rb.getString("JiHuo")%></span>
										</span>
									</div>
									<div v-else class="shibai">
										<span class="el-icon el-icon-status-active">
											<span class="f12 ml5"><%= rb.getString("QuJiHuo")%></span>
										</span>
									</div>
								</template>
								<template v-if="col.field == 'channelType'">
									<div v-if="scope.row.cellType == 'TC'">
										<div  v-if="!['','NULL','null',null,undefined].includes(scope.row.channelType)" class="cellItemBoxCls">
											<span>{{scope.row.channelType[0]}}</span>
											<el-popover title="All" popper-class="allCellPopoverClass">
												<span style="color:#4d84ff;" slot="reference">
													[ <span v-html="scope.row.channelType.length"></span> 
													<i class="el-icon el-icon-common-arrow-down" style="font-size: 12px; zoom: 0.7;"></i>]
												</span>
												<div class="allCellListCls">
													<i class="el-icon el-icon-close" style="position: absolute; top: -30px; right: 0px;" onclick="document.body.click()"></i>
													<div class="allCellListCls-item" v-for="(item,index) in scope.row.channelType">
														<div>Cell{{index+1}}:</div>
														<div>{{item}}</div>
													</div>
												</div>
											</el-popover>
										</div>
									</div>
									<div v-if="scope.row.cellType !== 'TC'">
										{{scope.row.channelType}}
									</div>
								</template>
								<template v-if="col.field == 'frequency'">
									<div v-if="scope.row.cellType == 'TC'">
										<div  v-if="!['','NULL','null',null,undefined].includes(scope.row.frequency)" class="cellItemBoxCls">
											<span class="cellItemOmitCls">
												{{scope.row.frequency[0]}}
											</span>
											<el-popover title="All" popper-class="allCellPopoverClass">
												<span style="color:#4d84ff;" slot="reference">
													[ <span v-html="scope.row.frequency.length"></span> 
													<i class="el-icon el-icon-common-arrow-down" style="font-size: 12px; zoom: 0.7;"></i>]
												</span>
												<div class="allCellListCls">
													<i class="el-icon el-icon-close" style="position: absolute; top: -30px; right: 0px;" onclick="document.body.click()"></i>
													<div class="allCellListCls-item" v-for="(item,index) in scope.row.frequency">
														<div>Cell{{index+1}}:</div>
														<div>{{item}}</div>
													</div>
												</div>
											</el-popover>
										</div>
									</div>
									<div v-if="scope.row.cellType !== 'TC'">
										{{scope.row.frequency}}
									</div>
								</template>
								<template v-if="col.field == 'grantExpireTime'">
									<div v-if="scope.row.cellType == 'TC'">
										<div  v-if="!['','NULL','null',null,undefined].includes(scope.row.grantExpireTime)" class="cellItemBoxCls">
											<span class="cellItemOmitCls">
												{{scope.row.grantExpireTime[0]}}
											</span>
											<el-popover title="All" popper-class="allCellPopoverClass">
												<span style="color:#4d84ff;" slot="reference">
													[ <span v-html="scope.row.grantExpireTime.length"></span> 
													<i class="el-icon el-icon-common-arrow-down" style="font-size: 12px; zoom: 0.7;"></i>]
												</span>
												<div class="allCellListCls">
													<i class="el-icon el-icon-close" style="position: absolute; top: -30px; right: 0px;" onclick="document.body.click()"></i>
													<div class="allCellListCls-item" v-for="(item,index) in scope.row.grantExpireTime">
														<div>Cell{{index+1}}:</div>
														<div>{{item}}</div>
													</div>
												</div>
											</el-popover>
										</div>
									</div>
									<div v-if="scope.row.cellType !== 'TC'">
										{{scope.row.grantExpireTime}}
									</div>
								</template>
								<template v-if="col.field == 'transmitExpireTime'">
									<div v-if="scope.row.cellType == 'TC'">
										<div  v-if="!['','NULL','null',null,undefined].includes(scope.row.transmitExpireTime)" class="cellItemBoxCls">
											<span class="cellItemOmitCls">
												{{scope.row.transmitExpireTime[0]}}
											</span>
											<el-popover title="All" popper-class="allCellPopoverClass">
												<span style="color:#4d84ff;" slot="reference">
													[ <span v-html="scope.row.transmitExpireTime.length"></span> 
													<i class="el-icon el-icon-common-arrow-down" style="font-size: 12px; zoom: 0.7;"></i>]
												</span>
												<div class="allCellListCls">
													<i class="el-icon el-icon-close" style="position: absolute; top: -30px; right: 0px;" onclick="document.body.click()"></i>
													<div class="allCellListCls-item" v-for="(item,index) in scope.row.transmitExpireTime">
														<div>Cell{{index+1}}:</div>
														<div>{{item}}</div>
													</div>
												</div>
											</el-popover>
										</div>
									</div>
									<div v-if="scope.row.cellType !== 'TC'">
										{{scope.row.transmitExpireTime}}
									</div>
								</template>
								<template v-if="col.field == 'cbsdId'">
									<div v-if="scope.row.cellType == 'TC'">
										<div  v-if="!['','NULL','null',null,undefined].includes(scope.row.cbsdId)" class="cellItemBoxCls">
											<span class="cellItemOmitCls">
												{{scope.row.cbsdId[0]}}
											</span>
											<el-popover title="All" popper-class="allCellPopoverClass">
												<span style="color:#4d84ff;" slot="reference">
													[ <span v-html="scope.row.cbsdId.length"></span> 
													<i class="el-icon el-icon-common-arrow-down" style="font-size: 12px; zoom: 0.7;"></i>]
												</span>
												<div class="allCellListCls">
													<i class="el-icon el-icon-close" style="position: absolute; top: -30px; right: 0px;" onclick="document.body.click()"></i>
													<div class="allCellListCls-item" v-for="(item,index) in scope.row.cbsdId">
														<div>Cell{{index+1}}:</div>
														<div>{{item}}</div>
													</div>
												</div>
											</el-popover>
										</div>
									</div>
									<div v-if="scope.row.cellType !== 'TC'">
										{{scope.row.cbsdId}}
									</div>
								</template>
								<template v-if="col.field == 'maxEirp'">
									<div v-if="scope.row.cellType == 'TC'">
										<div  v-if="!['','NULL','null',null,undefined].includes(scope.row.maxEirp)" class="cellItemBoxCls">
											<span>{{scope.row.maxEirp[0]}}</span>
											<el-popover title="All" popper-class="allCellPopoverClass">
												<span style="color:#4d84ff;" slot="reference">
													[ <span v-html="scope.row.maxEirp.length"></span> 
													<i class="el-icon el-icon-common-arrow-down" style="font-size: 12px; zoom: 0.7;"></i>]
												</span>
												<div class="allCellListCls">
													<i class="el-icon el-icon-close" style="position: absolute; top: -30px; right: 0px;" onclick="document.body.click()"></i>
													<div class="allCellListCls-item" v-for="(item,index) in scope.row.maxEirp">
														<div>Cell{{index+1}}:</div>
														<div>{{item}}</div>
													</div>
												</div>
											</el-popover>
										</div>
									</div>
									<div v-if="scope.row.cellType !== 'TC'">
										{{scope.row.maxEirp}}
									</div>
								</template>
								<template v-if="col.field == 'grants'">
									<div v-if="scope.row.cellType == 'TC'">
										<div  v-if="!['','NULL','null',null,undefined].includes(scope.row.grants)" class="cellItemBoxCls">
											<span class="cellItemOmitCls">
												{{scope.row.grants[0]}}
											</span>
											<el-popover title="All" popper-class="allCellPopoverClass">
												<span style="color:#4d84ff;" slot="reference">
													[ <span v-html="scope.row.grants.length"></span> 
													<i class="el-icon el-icon-common-arrow-down" style="font-size: 12px; zoom: 0.7;"></i>]
												</span>
												<div class="allCellListCls">
													<i class="el-icon el-icon-close" style="position: absolute; top: -30px; right: 0px;" onclick="document.body.click()"></i>
													<div class="allCellListCls-item" v-for="(item,index) in scope.row.grants">
														<div>Cell{{index+1}}:</div>
														<div>{{item}}</div>
													</div>
												</div>
											</el-popover>
										</div>
									</div>
									<div v-if="scope.row.cellType !== 'TC'">
										{{scope.row.grants}}
										<el-popover 
											v-if="scope.row.grants != '' && scope.row.grants != undefined && scope.row.grants != null"  
											trigger="click" width='300' 
											popper-class="grantsPopoverClass"
											@show="grantsPopoverShow(scope.row)" 
											@hide="grantsPopoverHide" 
											>
											<span v-if="scope.row.grants != '' && scope.row.grants != undefined && scope.row.grants != null" style="margin-left:5px;" class="el-icon el-icon-circle-down grantsIcon" slot="reference"></span>
											<div class="grantsItemBoxCls" v-loading="grantsPopoverLoading" >
												<div class="grantsItemCls">
													<div><%=rb.getString("YuanShiShouQuan")%><%=rb.getString("MaoHao")%></div>
													<div>
														<div v-for="item in grantsData.grantList">{{item}}</div>
													</div>
												</div>
												<div class="grantsItemCls">
													<div><%=rb.getString("LinShiShouQuan")%><%=rb.getString("MaoHao")%></div>
													<div>
														<div v-for="item in grantsData.temGrantList">{{item}}</div>
													</div>
												</div>
											</div>
										</el-popover>
									</div>
									
								</template>
								<template v-if="!['serialNumber','state','cpiState','opStatus','hostName','channelType','maxEirp','cbsdId','grants','frequency','grantExpireTime','transmitExpireTime','directMode'].includes(col.field)">
									{{ scope.row[col.field] }}
								</template>
							</template>
						</el-table-column>
					</el-ctable>
				</el-tab-pane>
				<!--CBSD gNB -->
				<el-tab-pane v-if="writableMap['CODE_GNB_MONITOR'] != undefined && activeName == 'gNB'" label='<%=rb.getString("gNB")%>' name="gNB"> <!--:url="gnbUrl" :data="gnbTableData"  -->
					<el-ctable id="gnbSASMonitorTable" ref="gnbTable" 
						:url="gnbUrl"  
						:limit="limitBatch"
						@sort-change="sortChangeGnb"
						:row-key="'small_cell_code'" 
						:query-params="params_sasGnb" 
						@selection-change='batchSelect' 
						@load-success="tableLoadSuccess" >
						<!-- 高级查询 -- gNB-->
						<template slot="toolbar">
							<div class="toolbarHeadBtnBoxCls">
								<div v-show="optBtnShow" class="selectBlukBoxCls">
									<div class="selectMain">
										 <div class="bulkSelectBtnBoxCls"  @click="openBulkSelectTable">
                                            <span class="el-icon-selected el-icon"></span>
                                            <span class="bulkSelectNumBoxCls">( {{selectionData.length}} )</span>
                                        </div>
										<div class="selectTableBoxCls" style="position: absolute;top: 38px;left: 0px;" v-show="gnbBulkSelectShow">
											<div class="selectBoxTitle">
												<span><%=rb.getString("YiXuan")%></span>
												<span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="closeBulkSelectTable()"></span>
											</div>
											<div class="selectBoxMain">
												<div class="tableInfoCls">
													<div class="tableInfoHeader">
														<div><%=rb.getString("YiXuanSheBei")%></div>
														<div @click="clearBulkSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span>Clear</div>
													</div>
													<el-ctable 
														id="bulkSelectTable" 
														ref="bulkSelectTable" 
														:data="selectionData" 
														:showHeader="false"
														:rownumber="false"
														:front-pagination="true"
														:row-key="'small_cell_code'"
														height="270px" pagination="true" >
														<el-table-column prop="serialNumber" v-if="false"></el-table-column>
														<el-table-column width="588">
															<template slot-scope="scope" >
																<div class="tableItemCls">
																	<span>{{scope.row.serialNumber}}</span>
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
								<div v-show="optBtnShow" :class="selectionData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="setInstall">
									<span class="el-icon el-icon-operation-settings"></span>
									<span><%=rb.getString("SheZhi")%></span>
								</div>
								<div v-show="optBtnShow" :class="selectionData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="batchCbrsas('on')">
									<span class="el-icon el-icon-status-enable"></span>
									<span>SAS ON</span>
								</div>
								<div v-show="optBtnShow" :class="selectionData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="batchCbrsas('off')">
									<span class="el-icon el-icon-status-disable"></span>
									<span>SAS OFF</span>
								</div>
							</div>
							<div style="position:relative;">
								<div id="tableHeadQuery" class="tableHeadQueryBoxCls">
									<div class="headQueryBox">
										<div class="queryGroup">
											<el-input v-model="gnb_search_text" @keyup.enter.native="query" @focus="queryInputFocus" @blur="queryInputBlur" :placeholder='placeholderText' style="width:260px;"></el-input>
											<i @click='query' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
										</div>
									</div>
									<el-popfilter style="margin: 0 5px;"
										type="single"
										label='<%=rb.getString("ShiFouJiHuo") %>'
										v-model="queryForm.opStatus"
										:list="opStatusList"
										@check-change="advanceQuery">
									</el-popfilter>
									<el-popfilter style="margin: 0 5px;"
										type="single"
										label='<%=rb.getString("ShiJiLeiXing") %>'
										v-model="queryForm.cbsdType"
										:list="cbsdTypeList"
										@check-change="advanceQuery">
									</el-popfilter>
									<div class="advancedQueryItemBox"  style="background: #FFF;margin-right:10px;" @click="clearFilterClick">
										<%=rb.getString("QingKongShaiXuan")%>
									</div>
									<!-- SAS 数据统计-->
									<div class="sasStatistics">
										<div class="enableStatistics">		
											<span class="enableTitleStatistics titleCommon enableBg"><%=rb.getString("SASKaiGuan") %></span>
											<span class="enableNumStatistics titleCommon">{{sasEnableCount}}/{{totalCount}}</span>
										</div>
										<div class="enableStatistics">		
											<span class="enableTitleStatistics titleCommon authorizedBg">Authorized</span>
											<span class="enableNumStatistics titleCommon">{{authorizedCount}}/{{totalCount}}</span>
										</div>
										<div class="enableStatistics">		
											<span class="enableTitleStatistics titleCommon grantSuspendedBg">Grant-Suspended</span>
											<span class="enableNumStatistics titleCommon">{{grantSuspendedCount}}/{{totalCount}}</span>
										</div>
									</div>
								</div>
								<span class="el-icon-operation-settings el-icon" 
								style="position:absolute;left:15px;bottom:-60px;z-index:99;width:30px;height: 40px;box-shadow:none;" 
								@click="gnbColumnSetting"></span>
							</div>
							<div class="showHideItem" id="gnb_sas_column_setting">
								<div class="select-all-cls" style="padding-left: 10px;">
									<el-checkbox :indeterminate="!gnbDragAll" v-model="gnbDragAll" @change="gnbDragAllChange"></el-checkbox> 
									<span><%=rb.getString("QuanXuan")%></span>
								</div>
								<el-checkbox-group v-model="gnbDragCol">
									<draggable
										class="list-group"
										v-model="gnbSortColumns"
										v-bind="dragOptions">
										<transition-group type="transition" :name="!gnbDrag? 'flip-list':null">
											<div v-for="(col,idx) in gnbSortColumns" :key="col.field" class="list-group-item">
												<el-checkbox :label="col.field" :key="col.field" :disabled="col.disabled">{{col.label}} </el-checkbox>
												<span v-if="false" style="position: absolute;right: 5px;top: 5px;">{{idx+1}}</span>
											</div>
										</transition-group>
									</draggable>
								</el-checkbox-group>

								<div class="windowButtonGroup" style="float:none !important;margin:20px 0 20px 30px">			
									<a class="linkbutton linkbutton_trend" @click="gnbColumnConfig"><span><%=rb.getString("QueDing")%></span></a>
									<a class="linkbutton linkbutton_nowanna" @click="gnbCancelConfig"><span><%=rb.getString("QuXiao")%></span></a>
								</div>
							</div>
						</template>
						
						<el-table-column label='' width="50" type="selection" :reserve-selection="true" :selectable = "isDisabled" v-if="optBtnShow"></el-table-column>
						<!-- 主列表 -->
						<el-table-column label='' width="90" prop="">
							<template slot-scope="scope">
								<!-- 如果是辅助小区，增加 carrierType字段，carrierType字段值为'scell'，表格操作项及SAS开关禁止点击 -->
								<div v-if="scope.row.carrierType == 'scell'" class="sasMonitorOptCls">
									<div class="el-icon el-icon-operation-settings disabled" title="<%=rb.getString("SheZhi") %>"></div>
									<div v-if="scope.row.cbsdType =='virtual'&&optBtnShow" class="el-icon el-icon-operation-delete disabled" title="<%=rb.getString("ShanChu") %>"></div>
									<div class="el-icon el-icon-operation-details" title="<%=rb.getString("RiZhi") %>" @click="sasLogs(scope.row)"></div>
								</div>
								<div v-else class="sasMonitorOptCls">
									<div class="el-icon el-icon-operation-settings" title="<%=rb.getString("SheZhi") %>" @click="goProperties(scope.row)"></div>
									<div v-if="scope.row.cbsdType =='virtual'&&optBtnShow" class="el-icon el-icon-operation-delete" title="<%=rb.getString("ShanChu") %>" @click="deleteVirtual(scope.row)"></div>
									<div class="el-icon el-icon-operation-details" title="<%=rb.getString("RiZhi") %>" @click="sasLogs(scope.row)"></div>
								</div>
							</template>
						</el-table-column>
						
						<el-table-column label='<%=rb.getString("SASKaiGuan") %>' width="110" prop="sasEnable" sortable header-align="left" align="center">
							<template slot-scope="scope" v-if="scope.row">
								<!-- 如果是辅助小区，增加 carrierType字段，carrierType字段值为'scell'，表格操作项及SAS开关禁止点击 -->
								<template v-if="!optBtnShow">
									<div v-if="scope.row.sasEnable == 'on' || scope.row.sasEnable == 'off'">
										<el-switch v-model="scope.row.sasEnable" style="height: 18px;margin-left: 10px;" :disabled="true"
											active-color="#4D84FF"
											active-value="on"
											inactive-value="off">									
										</el-switch>
									</div>
									<div v-else> -- </div>
								</template>
								<template v-else>
									<!-- 根据在线不在线状态判断显示 SAS 开关的状态 -->					
									<div v-if="scope.row.connection_status != 'Off'">
										<div v-if="scope.row.sasEnable == 'on' || scope.row.sasEnable == 'off'" style="display:flex;">
											<div v-if="scope.row.carrierType == 'scell'">
												<el-switch :ref="scope.row.serialNumber" v-model="scope.row.sasEnable" style="height: 18px;margin-left: 10px;" :disabled="true"
													active-color="#4D84FF"
													active-value="on"
													inactive-value="off">
												</el-switch>
											</div>
											<div v-else style="position:relative;">
												<el-switch :ref="scope.row.serialNumber" v-model="scope.row.sasEnable" style="height: 18px;margin-left: 10px;"
													active-color="#4D84FF"
													active-value="on"
													inactive-value="off">
												</el-switch>
												<div class="sasEnableClickBox" @click="sasEnabelChange(scope.row)"></div>
											</div>
											<div v-if="scope.row.sasEnable == 'off' && scope.row.scheduleTime">
												<el-popover style='word-wrap:break-word' placement='bottom' trigger='hover' width='260'>
													<div style="padding:20px;width:200px;height:60px;position:relative;">
														<div style="margin-bottom:10px;">
															<div><%= rb.getString("ZhiXingShiJian")%>:</div>
															<div style="padding-left:20px;">{{scope.row.scheduleTime}}</div>
														</div>
													</div>
													<span slot='reference' style="margin-left:10px;" class="el-icon el-icon-operation-time"></span>
												</el-popover>
											</div>
										</div>
										<div v-else> --</div>
									</div>
									
									<div v-else-if="scope.row.connection_status == 'Off'" style="display:flex;">
										<div v-if="scope.row.sasEnable == 'on' || scope.row.sasEnable == 'off'">
											<div v-if="scope.row.sasEnable == 'on'"  style="position:relative;">
												<el-switch :ref="scope.row.serialNumber" v-model="scope.row.sasEnable" style="height: 18px;margin-left: 10px;"
													:disabled="scope.row.cbsdType =='virtual'"
													active-color="#4D84FF"
													active-value="on"
													inactive-value="off">
												</el-switch>
												<div v-if="scope.row.cbsdType == 'virtual'" class="disabled"></div>
												<div v-if="scope.row.cbsdType != 'virtual'" class="sasEnableClickBox" @click="sasEnabelChange(scope.row)"></div>
											</div>
											<div v-if="scope.row.sasEnable == 'off'">
												<el-switch v-model="scope.row.sasEnable" style="height: 18px;margin-left: 10px;" :disabled="true"
													active-color="#4D84FF"
													active-value="on"
													inactive-value="off">									
												</el-switch>
											</div>
										</div>
										<div v-else> -- </div>
										<div v-if="scope.row.sasEnable == 'off' && scope.row.scheduleTime">
											<el-popover style='word-wrap:break-word' placement='bottom' trigger='hover' width='260'>
												<div style="padding:20px;width:200px;height:50px;position:relative;">
													<div style="margin-bottom:20px;">
														<div><%= rb.getString("ZhiXingShiJian")%>:</div>
														<div style="padding-left:20px;">{{scope.row.scheduleTime}}</div>
													</div>
												</div>
												<span slot='reference' style="margin-left:10px;" class="el-icon el-icon-operation-time"></span>
											</el-popover>
										</div>								
									</div>
									
									<div v-else></div>
								</template>
							</template>
						</el-table-column>
									
						<el-table-column prop="connection_status" sortable min-width="50" width="50" >
							<template slot-scope="scope">
								<div :class="{
									'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
									'':scope.row.have_connected==2,
									'conn_exc':scope.row.connection_status=='Exception',
									'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:20px;'></div>
							</template>
						</el-table-column>

						<el-table-column v-for="col in gnbColumns" :prop="col.field" :label="col.label" :sortable="col.sortable" :min-width="col.width" v-if="gnbShowColums.includes(col.field)">
							<template slot-scope="scope">
								<template v-if="col.field == 'serialNumber'">
									{{scope.row.serialNumber}}<div class="virtualIconClass" v-if="scope.row.cbsdType =='virtual'">Virtual</div>
								</template>
								<template v-if="col.field == 'directMode'">
									<div v-if="scope.row.directMode =='on'"><%=rb.getString("ZhiLianMoShi") %></div>
									<div v-if="scope.row.directMode =='off'"><%=rb.getString("ShuJuChuLiMoShi") %></div>
								</template>
								<template v-if="col.field == 'state'">
									<span v-html="sasStateFmt(scope.row.state)"></span>
								</template>
								<template v-if="col.field == 'opStatus'">
									<div v-if="scope.row.opStatus == '1'" class="chengong">
										<span class="el-icon el-icon-status-active">
											<span class="f12 ml5"><%= rb.getString("JiHuo")%></span>
										</span>
									</div>
									<div v-else class="shibai">
										<span class="el-icon el-icon-status-active">
											<span class="f12 ml5"><%= rb.getString("QuJiHuo")%></span>
										</span>
									</div>
								</template>
								<template v-if="col.field == 'grants'">
									<div>
										{{scope.row.grants}}
										<el-popover 
											v-if="scope.row.grants != '' && scope.row.grants != undefined && scope.row.grants != null"  
											trigger="click" width='300' 
											popper-class="grantsPopoverClass"
											@show="grantsPopoverShow(scope.row)" 
											@hide="grantsPopoverHide" 
											>
											<span v-if="scope.row.grants != '' && scope.row.grants != undefined && scope.row.grants != null" style="margin-left:5px;" class="el-icon el-icon-circle-down grantsIcon" slot="reference"></span>
											<div class="grantsItemBoxCls" v-loading="grantsPopoverLoading" >
												<div class="grantsItemCls">
													<div><%=rb.getString("YuanShiShouQuan")%><%=rb.getString("MaoHao")%></div>
													<div>
														<div v-for="item in grantsData.grantList">{{item}}</div>
													</div>
												</div>
												<div class="grantsItemCls">
													<div><%=rb.getString("LinShiShouQuan")%><%=rb.getString("MaoHao")%></div>
													<div>
														<div v-for="item in grantsData.temGrantList">{{item}}</div>
													</div>
												</div>
											</div>
										</el-popover>
									</div>
									
								</template>
								<template v-if="!['serialNumber','state','opStatus','grants','directMode'].includes(col.field)">
									{{ scope.row[col.field] }}
								</template>
							</template>
						</el-table-column>
					</el-ctable>
				</el-tab-pane>
				<!--CBSD CPE -->
				<el-tab-pane v-if="writableMap['CODE_CPE_MONITOR'] != undefined && activeName == 'CPE'" label="CPE" name="CPE"><!--:url="cpeUrl" :data="cpeTableData" -->
					<el-ctable id="cpeSASMonitorTable" ref="cpeTable" :url="cpeUrl"  @sort-change="sortChangeCPE" :limit="limitBatch"
						:row-key="'small_cell_code'" :query-params="params_sasCpe" @selection-change='batchSelect' @load-success="tableLoadSuccess">
						<!-- 高级查询 -- CPE-->
						<template slot="toolbar" style="padding:0px;">
							<div class="toolbarHeadBtnBoxCls">
								<div v-show="optBtnShow" class="selectBlukBoxCls">
									<div class="selectMain">
										 <div class="bulkSelectBtnBoxCls"  @click="openBulkSelectTable">
                                            <span class="el-icon-selected el-icon"></span>
                                            <span class="bulkSelectNumBoxCls">( {{selectionData.length}} )</span>
                                        </div>
										<div class="selectTableBoxCls" style="position: absolute;top: 38px;left: 0px;" v-show="cpeBulkSelectShow">
											<div class="selectBoxTitle">
												<span><%=rb.getString("YiXuan")%></span>
												<span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="closeBulkSelectTable()"></span>
											</div>
											<div class="selectBoxMain">
												<div class="tableInfoCls">
													<div class="tableInfoHeader">
														<div><%=rb.getString("YiXuanSheBei")%></div>
														<div @click="clearBulkSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span>Clear</div>
													</div>
													<el-ctable 
														id="bulkSelectTable" 
														ref="bulkSelectTable" 
														:data="selectionData" 
														:showHeader="false"
														:rownumber="false"
														:front-pagination="true"
														:row-key="'small_cell_code'"
														height="270px" pagination="true" >
														<el-table-column prop="serialNumber" v-if="false"></el-table-column>
														<el-table-column width="588">
															<template slot-scope="scope" >
																<div class="tableItemCls">
																	<span>{{scope.row.serialNumber}}</span>
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
								<div v-show="optBtnShow" :class="selectionData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="setInstall">
									<span class="el-icon el-icon-operation-settings"></span>
									<span><%=rb.getString("SheZhi")%></span>
								</div>
								<div v-show="optBtnShow" :class="selectionData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="batchCbrsas('on')">
									<span class="el-icon el-icon-status-enable"></span>
									<span>SAS ON</span>
								</div>
								<div v-show="optBtnShow" :class="selectionData.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="batchCbrsas('off')">
									<span class="el-icon el-icon-status-disable"></span>
									<span>SAS OFF</span>
								</div>
							</div>
							<div style="position:relative;">
								<div id="tableHeadQuery" class="tableHeadQueryBoxCls">
									<div class="headQueryBox">
										<div class="queryGroup">
											<el-input v-model="cpe_search_text" @keyup.enter.native="query" @focus="queryInputFocus" @blur="queryInputBlur" :placeholder='placeholderText' style="width:260px;"></el-input>
											<i @click='query' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
										</div>
									</div>
									<el-popfilter style="margin: 0 5px;"
										type="single"
										label='<%=rb.getString("ShiJiLeiXing") %>'
										v-model="queryForm.cbsdType"
										:list="cbsdTypeList"
										@check-change="advanceQuery">
									</el-popfilter>
									<div class="advancedQueryItemBox"  style="background: #FFF;margin-right:10px;" @click="clearFilterClick">
										<%=rb.getString("QingKongShaiXuan")%>
									</div>
									<!-- SAS 数据统计-->
									<div class="sasStatistics">
										<div class="enableStatistics">		
											<span class="enableTitleStatistics titleCommon enableBg"><%=rb.getString("SASKaiGuan") %></span>
											<span class="enableNumStatistics titleCommon">{{sasEnableCount}}/{{totalCount}}</span>
										</div>
										<div class="enableStatistics">		
											<span class="enableTitleStatistics titleCommon authorizedBg">Authorized</span>
											<span class="enableNumStatistics titleCommon">{{authorizedCount}}/{{totalCount}}</span>
										</div>
										<div class="enableStatistics">		
											<span class="enableTitleStatistics titleCommon grantSuspendedBg">Grant-Suspended</span>
											<span class="enableNumStatistics titleCommon">{{grantSuspendedCount}}/{{totalCount}}</span>
										</div>
									</div>
								</div>
								<span class="el-icon-operation-settings el-icon" 
								style="position:absolute;left:15px;bottom:-60px;z-index:99;width:30px;height: 40px;box-shadow:none;" 
								@click="cpeColumnSetting"></span>
							</div>
							<div class="showHideItem" id="cpe_sas_column_setting">
								<div class="select-all-cls" style="padding-left: 10px;">
									<el-checkbox :indeterminate="!cpeDragAll" v-model="cpeDragAll" @change="cpeDragAllChange"></el-checkbox> 
									<span><%=rb.getString("QuanXuan")%></span>
								</div>
								<el-checkbox-group v-model="cpeDragCol">
									<draggable
										class="list-group"
										v-model="cpeSortColumns"
										v-bind="dragOptions">
										<transition-group type="transition" :name="!drag? 'flip-list':null">
											<div v-for="(col,idx) in cpeSortColumns" :key="col.field" class="list-group-item">
												<el-checkbox :label="col.field" :key="col.field" :disabled="col.disabled">{{col.label}} </el-checkbox>
												<span v-if="false" style="position: absolute;right: 5px;top: 5px;">{{idx+1}}</span>
											</div>
										</transition-group>
									</draggable>
								</el-checkbox-group>

								<div class="windowButtonGroup" style="float:none !important;margin:20px 0 20px 30px">			
									<a class="linkbutton linkbutton_trend" @click="cpeColumnConfig"><span><%=rb.getString("QueDing")%></span></a>
									<a class="linkbutton linkbutton_nowanna" @click="cpeCancelConfig"><span><%=rb.getString("QuXiao")%></span></a>
								</div>
							</div>
						</template>
						
						<el-table-column label='' width="50" type="selection" :reserve-selection="true" :selectable = "isDisabledCpe" v-if="optBtnShow"></el-table-column>
						<!-- 主列表 -->
						<el-table-column label='' width="90" prop="">
							<template slot-scope="scope">
								<div class="el-icon el-icon-operation-settings" title="<%=rb.getString("SheZhi") %>" @click="goProperties(scope.row)"></div>
								<div v-if="scope.row.cbsdType =='virtual' && optBtnShow" class="el-icon el-icon-operation-delete" title="<%=rb.getString("ShanChu") %>" @click="deleteVirtual(scope.row)"></div>
								<div class="el-icon el-icon-operation-details" title="<%=rb.getString("RiZhi") %>" @click="sasLogs(scope.row)"></div>
							</template>
						</el-table-column>
						<el-table-column label='<%=rb.getString("SASKaiGuan") %>' width="110" prop="sasEnable" sortable header-align="left" align="center">
							<template slot-scope="scope" v-if="scope.row">
								<template v-if="!optBtnShow">
									<div v-if="scope.row.sasEnable == 'on' || scope.row.sasEnable == 'off'">
										<el-switch v-model="scope.row.sasEnable" style="height: 18px;margin-left: 10px;" :disabled="true"
											active-color="#4D84FF"
											active-value="on"
											inactive-value="off">									
										</el-switch>
									</div>
									<div v-else> -- </div>
								</template>
								<template v-else>
									<!-- 根据在线不在线状态判断显示 SAS 开关的状态 -->					
									<div v-if="scope.row.connection_status != 'Off'" style="display:flex;">
										<div v-if="scope.row.sasEnable == 'on' || scope.row.sasEnable == 'off'" style="position:relative;">
											<el-switch :ref="scope.row.serialNumber" v-model="scope.row.sasEnable" style="height: 18px;margin-left: 10px;"
												active-color="#4D84FF"
												active-value="on"
												inactive-value="off">	
											</el-switch>
											<div class="sasEnableClickBox" @click="sasEnabelChange(scope.row)"></div>
										</div>
										<div v-else> -- </div>
										<div v-if="scope.row.sasEnable == 'off' && scope.row.scheduleTime">
											<el-popover style='word-wrap:break-word' placement='bottom' trigger='hover' width='260'>
												<div style="padding:20px;width:200px;height:60px;position:relative;">
													<div style="margin-bottom:10px;">
														<div><%= rb.getString("ZhiXingShiJian")%>:</div>
														<div style="padding-left:20px;">{{scope.row.scheduleTime}}</div>
													</div>
												</div>
												<span slot='reference' style="margin-left:10px;" class="el-icon el-icon-operation-time"></span>
											</el-popover>
										</div>
									</div>
									
									<div v-else-if="scope.row.connection_status == 'Off'" style="position:relative;display:flex;">
										<div v-if="scope.row.sasEnable == 'on' || scope.row.sasEnable == 'off'">
											<div v-if="scope.row.sasEnable == 'on'"  style="position:relative;">
												<el-switch :ref="scope.row.serialNumber" v-model="scope.row.sasEnable" style="height: 18px;margin-left: 10px;"
													:disabled="scope.row.cbsdType =='virtual'"
													active-color="#4D84FF"
													active-value="on"
													inactive-value="off">
												</el-switch>
												<div v-if="scope.row.cbsdType == 'virtual'" class="disabled"></div>
												<div v-if="scope.row.cbsdType != 'virtual'" class="sasEnableClickBox" @click="sasEnabelChange(scope.row)"></div>
											</div>
											<div v-if="scope.row.sasEnable == 'off'">
												<el-switch v-model="scope.row.sasEnable" style="height: 18px;margin-left: 10px;" :disabled="true"
													active-color="#4D84FF"
													active-value="on"
													inactive-value="off">									
												</el-switch>
											</div>
										</div>
										<div v-else> -- </div>	
										<div v-if="scope.row.sasEnable == 'off' && scope.row.scheduleTime">
											<el-popover style='word-wrap:break-word' placement='bottom' trigger='hover' width='260'>
												<div style="padding:20px;width:200px;height:50px;position:relative;">
													<div style="margin-bottom:20px;">
														<div><%= rb.getString("ZhiXingShiJian")%>:</div>
														<div style="padding-left:20px;">{{scope.row.scheduleTime}}</div>
													</div>
												</div>
												<span slot='reference' style="margin-left:10px;" class="el-icon el-icon-operation-time"></span>
											</el-popover>
										</div>								
									</div>
									
									<div v-else></div>
								</template>
							</template>
						</el-table-column>
						<el-table-column prop="connection_status" sortable  min-width="50" width="50">
						
							<template slot-scope="scope">
								<div :class="{
									'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
									'':scope.row.have_connected==2,
									'conn_exc':scope.row.connection_status=='Exception',
									'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:20px;'></div>
							</template>
						</el-table-column>
						
						<el-table-column v-for="col in cpeColumns" :prop="col.field" :label="col.label" :sortable="col.sortable" :min-width="col.width" v-if="cpeShowColums.includes(col.field)">
							<template slot-scope="scope">
								<template v-if="col.field == 'serialNumber'">
									{{scope.row.serialNumber}}<div class="virtualIconClass" v-if="scope.row.cbsdType =='virtual'">Virtual</div>
								</template>
								<template v-if="col.field == 'directMode'">
									<div v-if="scope.row.directMode =='on'"><%=rb.getString("ZhiLianMoShi") %></div>
									<div v-if="scope.row.directMode =='off'"><%=rb.getString("ShuJuChuLiMoShi") %></div>
								</template>
								<template v-if="col.field == 'state'">
									<div class="sasStatedBox">
										<span v-if="scope.row.cpeSasStatusEqualOmc == '0'" class="el-icon el-icon-circle-warning" ></span>
										<span v-if="scope.row.state == '0'">Unregistered</span>
										<span v-if="scope.row.state == '1'">Registered</span>
										<span v-if="scope.row.state == '3'">Granted</span>
										<span v-if="scope.row.state == '4'">Grant Suspended</span>
										<span v-if="scope.row.state == '5'">Authorized</span>
										<span v-if="scope.row.state == '6'">Transmission</span>
									</div>
								</template>
								
								<template v-if="!['serialNumber','state','grants','directMode'].includes(col.field)">
									{{ scope.row[col.field] }}
								</template>
								<template v-if="col.field == 'grants'">
									
									<el-popover 
										v-if="scope.row.grants != '' && scope.row.grants != undefined && scope.row.grants != null"  
										trigger="click" width='300' 
										popper-class="grantsPopoverClass"
										@show="grantsPopoverShow(scope.row)" 
										@hide="grantsPopoverHide" 
										>
										<span v-if="scope.row.grants != '' && scope.row.grants != undefined && scope.row.grants != null" style="margin-left:5px;" class="el-icon el-icon-circle-down grantsIcon" slot="reference"></span>
										<div class="grantsItemBoxCls" v-loading="grantsPopoverLoading" >
											<div class="grantsItemCls">
												<div><%=rb.getString("YuanShiShouQuan")%><%=rb.getString("MaoHao")%></div>
												<div>
													<div v-for="item in grantsData.grantList">{{item}}</div>
												</div>
											</div>
											<div class="grantsItemCls">
												<div><%=rb.getString("BeiFenShouQuan")%><%=rb.getString("MaoHao")%></div>
												<div>
													<div v-for="item in grantsData.backupGrantList">{{item}}</div>
												</div>
											</div>
										</div>
									</el-popover>
								</template>
							</template>
						</el-table-column>
					</el-ctable>
				</el-tab-pane>
			</el-tabs>
		</div>
		<div class="importSasCertBoxCls" v-if="importSasCertshow == 'true'">
			<div class="rightOutBoxHeadCls">
				<span>{{rightBoxTitle}}</span>
				<span class="el-icon el-icon-close greyIcon" @click="rightBoxClose"></span>
			</div>
			<div class="rightItemMainBox" v-if="rightBoxType == 'sasCert'">
				<div style="margin-bottom:20px;color:#7A7992;"><%=rb.getString("CPIBuCunChuMiMa")%></div>
				<el-form label-position="top" ref="sasImportCertForm" :model='sasImportCertForm' :rules='sasImportCertFormRules'>
					 <el-form-item label="CPI  <%=rb.getString("ZhengShu")%>">
						<span slot="label" class="labelSlotCls">
							CPI <%=rb.getString("ZhengShu")%>
							<span>( Only .P12/.PEM are supported )</span>
						</span>
						<el-upload :on-success='checkFile' :on-change="fileChangePrivateKey" :show-file-list=false ref="uploadCert" :action="sasImportCertForm.uploadFileURL"
							:auto-upload="false">
							<el-input :readonly="true" :value="sasImportCertForm.fileName" class="w300">
								<a slot="append" class="el-icon el-icon-operation-import greyIcon" @click="fileSelectPrivateKey"></a>
							</el-input>
							<div slot="tip" class="el-form-item__error" v-show="sasImportCertForm.errorFlag">
								{{sasImportCertForm.errorMessage}}
							</div>
							<a slot="trigger" ref="file_up"></a>
						</el-upload>
					 </el-form-item>
					<el-form-item label="">
						<el-checkbox v-model="sasImportCertForm.p12" true-label="true" false-label="false" @change="p12CheckChange">P12</el-checkbox>
					</el-form-item>
					<el-form-item v-show="sasImportCertForm.p12 == 'true'" label="<%=rb.getString("MiMa")%>" prop="password">
						<el-password v-model="sasImportCertForm.password" size="mini" placeholder="" class="w300" maxlength='50' show-password></el-password>
						<el-input v-model="sasImportCertForm.password" style="display: none;"></el-input>
					</el-form-item>
					<el-form-item label="CPI ID" prop='cpiid'>
                        <el-input v-model="sasImportCertForm.cpiid" class="w300" maxlength='50'></el-input>
                    </el-form-item>
                    <el-form-item label="CPI  <%=rb.getString("MingCheng")%>" prop='cpiname'>
                        <el-input v-model="sasImportCertForm.cpiname" class="w300" maxlength='50'></el-input>
                    </el-form-item>
				</el-form>
			</div>
			<div class="rightItemMainBox" v-if="rightBoxType == 'settings'">
				<el-form label-position="top" :model="settingsEnbForm" ref="settingsEnbForm" >
					<el-form-item  label="" prop="sasAble" >
						<span style="margin-right:20px;font-size: 14px;"><%=rb.getString("SASZiDongZhuCeKaiGuan")%></span>
						<el-switch 
							:disabled="isabled"
							v-model="settingsEnbForm.sasAble"
							active-value="1"
							inactive-value="0"
							active-color="#4D84FF" 
							inactive-color="#BDC1C6"
						></el-switch>
					</el-form-item>
					<el-form-item  label="<%=rb.getString("SASTiGongShang")%>" prop="sasProvider" >
						<el-select v-model="settingsEnbForm.sasProvider" :disabled="isabled" @change="sasProviderChange">
							<el-option v-for="item in selectSasProvider" :key="item.value" :value="item.value" :label="item.label"></el-option>
						</el-select>
					</el-form-item>
					<el-form-item label="User ID" prop="sasUserId">
						<el-input v-model="settingsEnbForm.sasUserId" :disabled="isabled" placeholder='' ></el-input>
					</el-form-item>
				</el-form>
			</div>
			<div class="rightItemMainBox" v-if="rightBoxType == 'sasImport'">
				<el-form label-position="top" ref="sasImportForm" :model='sasImportForm' :rules='sasImportFormRules'>
					<div style="padding: 0 15px;">
						<el-form-item label="<%=rb.getString("ExcelFile")%> " style="margin-bottom:30px">
							<el-upload :on-success='sasImportCheckFile' :on-change="sasImportFileChange" :show-file-list=false ref="uploadSas" :action="sasImportForm.uploadFileURL"
								:auto-upload="false">
								<el-input :readonly="true" :value='sasImportFileName' class="w200">
									<i slot="append" class="el-icon el-icon-operation-import greyIcon" style="top:7px;" @click="sasImportFileSelect"></i>
								</el-input>
								<span class="clor9">.xls/.xlsx</span>
								<div slot="tip" class="el-form-item__error" v-show="!sasImportTypeFlag">
									<%=rb.getString("ZhiZhiChiExeclFile")%> 
								</div>
								<div slot="tip" class="el-form-item__error" v-show="sasImportSelectFlag">
									<%=rb.getString("QingXianXuanZeWenJian")%>
								</div>
								<div style="text-align:bottom">
									<span class="ml30" style="cursor: pointer"><i class="el-icon el-icon-common-download"></i><u @click='exportSasImportTemp'> <%=rb.getString("XiaZaiShiLiMoBan")%></u> </span>
								</div>
								<a slot="trigger" ref="file_up"></a>
							</el-upload>
						</el-form-item>
						<el-form-item label="User ID" prop="userid">
							<el-input v-model="sasImportForm.userid" class="w200" maxlength='50'></el-input>
						</el-form-item>
					</div>
				</el-form>
			</div>
			<div class="footer">
				<div class="lnkbuttonGroup" v-show="!isabled && rightBoxType == 'settings'" style="margin-left:20px;">
					<el-button  type="primary"  @click="submitSASSetting"><%=rb.getString("QueDing")%></el-button>
					<el-button  @click="rightBoxClose"><%=rb.getString("QuXiao")%></el-button>
				</div>
				<div class="lnkbuttonGroup" v-show="rightBoxType == 'sasCert'" style="margin-left:20px;">
					<el-button type="primary" @click="importSasCertSubmit"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="rightBoxClose"><%=rb.getString("QuXiao")%></el-button>
				</div>
				<div class="lnkbuttonGroup" v-show="rightBoxType == 'sasImport'" style="margin-left:20px;">
					<el-button type="primary" @click="sasImportSubmit"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="rightBoxClose"><%=rb.getString("QuXiao")%></el-button>
				</div>
			</div>
		</div>
	</div>
	<el-tslide class='tslide' ref="tslide" :url="tsslideUrl" :title="slideTitleView" :footer="false" :header='tsslideHeader' :position="tsslidePosition"
			:height="tsslideHeight" :modal='modal'  :width="tsslideWidth" @cancel='cancelImport' >
	</el-tslide>
			
	 <!-- 二级页面 -- insatall ,设置Enb和 过程  -->
	 <el-slide ref="slideView" :url="slideUrl" :title="slideTitleView" :footer="false" :header='slideHeader' :position="slidePosition" :force-position="true"
	    :height="slideHeight" :modal='modal' :width="slideWidth" @cancel='cancelViewSlide'>
	</el-slide>
	<!--二级页面 ———— 批量设置userid  -->
	<el-dialog :title='dialogTitle' :visible.sync="showWindowInfo" ref="windowDialog" :width="windowWidth" 
		:close-on-click-modal="false"  :url="dialogUrl" @close='closeDialog'  @success="openDialogSuc"  :append-to-body="true">
		<el-form label-position="left" :model="settingsInfo" :rules="settingsRules" ref="settingsForm" >
				<div class="form-group last userBox" >
					<div class="plr15">
						<el-form-item label="User ID" prop="userId" >
							<el-input v-model="settingsInfo.userId" placeholder='' class="w290" maxlength='48' style="margin-top:5px"></el-input>
						</el-form-item>
						<el-form-item label="Call Sign">
							<el-input v-model="settingsInfo.callSign" placeholder='' class="w290" style="margin-top:5px"></el-input>
						</el-form-item>
					</div>
				</div>
			</el-form>
			<div class="el-message-box__btns">
				<el-button @click="closeDialog"><%=rb.getString("QuXiao")%></el-button>
				<el-button  type="primary"  @click="sasBatchParams"><%=rb.getString("QueDing")%></el-button>
			</div>
	</el-dialog>
</div>
</div>
<script type="text/javascript">
var refreshTable;
if(window.sasMonitorPage) {
	try {
		window.sasMonitorPage.$destroy();
	}catch(e){}
}
window.sasMonitorPage = new Vue({
	el:'#sasMonitorPage',
	data(){
		var passwordValidator = (rule, value, callback) => {
				if(this.sasImportCertForm.p12 == 'true'){
					if (value === '') {
						callback('<%=rb.getString("QingShuRuMiMa") %>');
					} else {
						callback()
					}
				}else{
					callback()
				}
			},
		 	useridValidator = (rule, value, cb) => {
				if (value === '') {
					cb('User ID<%=rb.getString("BuNengWeiKong") %>');
				} else {
					cb()
				}
			};
		var codes = {
				enb: 'eNB',
				cpe: 'CPE',
				gnb: 'gNB'
			},
			netType = codes[sysMain.headType];

		return{
			showProps: ['serialNumber','directMode','hostName','state','cpiState','cell_ip'],
			dragCol: ['serialNumber','directMode','hostName','state','cpiState','cell_ip'],
			drag: false,
			columns: [
				{field: 'serialNumber', label: '<%=rb.getString("XiaoZhanBianMa") %>',sortable: true,disabled: true,width: 230},
				{field: 'hostName', label: '<%=rb.getString("HostName") %>',sortable: 'custom',disabled: true,width: 150},
				{field: 'cell_ip', label: 'IP',sortable: 'custom',disabled: true,width: 150},
				{field: 'directMode', label: '<%=rb.getString("MoShi") %>',sortable: false,disabled: true,width: 120},
				{field: 'state', label: '<%=rb.getString("SASZhuangTai") %>',sortable: 'custom',disabled: true,width: 120},
				{field: 'cpiState', label: '<%=rb.getString("CPIZhuangTai") %>',sortable: 'custom',disabled: true,width: 120},
				{field: 'category', label: '<%=rb.getString("SASLeiXing") %>',sortable: false,width: 80},
				{field: 'cbsdId', label: '<%=rb.getString("CBSDID") %>',sortable: false,width: 180},
				{field: 'channelType', label: '<%=rb.getString("XinDaoLeiXing") %>',sortable: false,width: 180},
				{field: 'grants', label: '<%=rb.getString("ShouQuan") %>',sortable: false,width: 200},
				{field: 'maxEirp', label: '<%=rb.getString("SASEIRP") %>',sortable: false,width: 180},
				{field: 'opStatus', label: '<%=rb.getString("ShiFouJiHuo") %>',sortable: false,width: 110},
				{field: 'frequency', label: '<%=rb.getString("PinLvHeZi") %>',sortable: false,width: 150},
				{field: 'grantExpireTime', label: '<%=rb.getString("grantGuoQiShiJian")%>',sortable: false,width: 170},
				{field: 'transmitExpireTime', label: '<%=rb.getString("transmitGuoQiShiJian")%>',sortable: false,width: 170},
				{field: 'supportSpec', label: '<%=rb.getString("WuXianDianJiShu")%>',sortable: false,width: 150},
				{field: 'maxTxPower', label: '<%=rb.getString("CPETxPower")%>',sortable: false,width: 150},
			],
			sortColumns: [
				{field: 'serialNumber', label: '<%=rb.getString("XiaoZhanBianMa") %>',sortable: true,disabled: true,width: 230},
				{field: 'hostName', label: '<%=rb.getString("HostName") %>',sortable: 'custom',disabled: true,width: 150},
				{field: 'cell_ip', label: 'IP',sortable: true,disabled: true,width: 150},
				{field: 'directMode', label: '<%=rb.getString("MoShi") %>',sortable: false,disabled: true,width: 120},
				{field: 'state', label: '<%=rb.getString("SASZhuangTai") %>',sortable: true,disabled: true,width: 120},
				{field: 'cpiState', label: '<%=rb.getString("CPIZhuangTai") %>',sortable: true,disabled: true,width: 120},
				{field: 'category', label: '<%=rb.getString("SASLeiXing") %>',width: 80},
				{field: 'cbsdId', label: '<%=rb.getString("CBSDID") %>',width: 180},
				{field: 'channelType', label: '<%=rb.getString("XinDaoLeiXing") %>',width: 180},
				{field: 'grants', label: '<%=rb.getString("ShouQuan") %>',width: 200},
				{field: 'maxEirp', label: '<%=rb.getString("SASEIRP") %>',width: 180},
				{field: 'opStatus', label: '<%=rb.getString("ShiFouJiHuo") %>',width: 110},
				{field: 'frequency', label: '<%=rb.getString("PinLvHeZi") %>',width: 150},
				{field: 'grantExpireTime', label: '<%=rb.getString("grantGuoQiShiJian")%>',width: 170},
				{field: 'transmitExpireTime', label: '<%=rb.getString("transmitGuoQiShiJian")%>',width: 170},
				{field: 'supportSpec', label: '<%=rb.getString("WuXianDianJiShu")%>',width: 150},
				{field: 'maxTxPower', label: '<%=rb.getString("CPETxPower")%>',width: 150},
			],
			origionSort: [],

			gnbShowProps: ['serialNumber','directMode','hostName','state','cpiState','cell_ip'],
			gnbDragCol: ['serialNumber','directMode','hostName','state','cpiState','cell_ip'],
			gnbDrag: false,
			gnbColumns: [
				{field: 'serialNumber', label: '<%=rb.getString("XiaoZhanBianMa") %>',sortable: true,disabled: true,width: 230},
				{field: 'hostName', label: '<%=rb.getString("HostName") %>',sortable: 'custom',disabled: true,width: 150},
				{field: 'cell_ip', label: 'IP',sortable: 'custom',disabled: true,width: 150},
				{field: 'directMode', label: '<%=rb.getString("MoShi") %>',sortable: false,disabled: true,width: 120},
				{field: 'state', label: '<%=rb.getString("SASZhuangTai") %>',sortable: 'custom',disabled: true,width: 120},
				{field: 'cpiState', label: '<%=rb.getString("CPIZhuangTai") %>',sortable: 'custom',disabled: true,width: 120},
				{field: 'category', label: '<%=rb.getString("SASLeiXing") %>',sortable: false,width: 80},
				{field: 'cbsdId', label: '<%=rb.getString("CBSDID") %>',sortable: false,width: 180},
				{field: 'channelType', label: '<%=rb.getString("XinDaoLeiXing") %>',sortable: false,width: 180},
				{field: 'grants', label: '<%=rb.getString("ShouQuan") %>',sortable: false,width: 200},
				{field: 'maxEirp', label: '<%=rb.getString("SASEIRP") %>',sortable: false,width: 180},
				{field: 'opStatus', label: '<%=rb.getString("ShiFouJiHuo") %>',sortable: false,width: 110},
				{field: 'frequency', label: '<%=rb.getString("PinLvHeZi") %>',sortable: false,width: 150},
				{field: 'grantExpireTime', label: '<%=rb.getString("grantGuoQiShiJian")%>',sortable: false,width: 170},
				{field: 'transmitExpireTime', label: '<%=rb.getString("transmitGuoQiShiJian")%>',sortable: false,width: 170},
				{field: 'supportSpec', label: '<%=rb.getString("WuXianDianJiShu")%>',sortable: false,width: 150},
				{field: 'maxTxPower', label: '<%=rb.getString("CPETxPower")%>',sortable: false,width: 150},
			],
			gnbSortColumns: [
				{field: 'serialNumber', label: '<%=rb.getString("XiaoZhanBianMa") %>',sortable: true,disabled: true,width: 230},
				{field: 'hostName', label: '<%=rb.getString("HostName") %>',sortable: 'custom',disabled: true,width: 150},
				{field: 'cell_ip', label: 'IP',sortable: true,disabled: true,width: 150},
				{field: 'directMode', label: '<%=rb.getString("MoShi") %>',sortable: false,disabled: true,width: 120},
				{field: 'state', label: '<%=rb.getString("SASZhuangTai") %>',sortable: true,disabled: true,width: 120},
				{field: 'cpiState', label: '<%=rb.getString("CPIZhuangTai") %>',sortable: true,disabled: true,width: 120},
				{field: 'category', label: '<%=rb.getString("SASLeiXing") %>',width: 80},
				{field: 'cbsdId', label: '<%=rb.getString("CBSDID") %>',width: 180},
				{field: 'channelType', label: '<%=rb.getString("XinDaoLeiXing") %>',width: 180},
				{field: 'grants', label: '<%=rb.getString("ShouQuan") %>',width: 200},
				{field: 'maxEirp', label: '<%=rb.getString("SASEIRP") %>',width: 180},
				{field: 'opStatus', label: '<%=rb.getString("ShiFouJiHuo") %>',width: 110},
				{field: 'frequency', label: '<%=rb.getString("PinLvHeZi") %>',width: 150},
				{field: 'grantExpireTime', label: '<%=rb.getString("grantGuoQiShiJian")%>',width: 170},
				{field: 'transmitExpireTime', label: '<%=rb.getString("transmitGuoQiShiJian")%>',width: 170},
				{field: 'supportSpec', label: '<%=rb.getString("WuXianDianJiShu")%>',width: 150},
				{field: 'maxTxPower', label: '<%=rb.getString("CPETxPower")%>',width: 150},
			],
			gnbOrigionSort: [],

			cpeShowProps: ['serialNumber','directMode','imsi','cpeName','state','cpiState'],
			cpeDragCol: ['serialNumber','directMode','imsi','cpeName','state','cpiState'],
			cpeColumns: [
				{field: 'serialNumber', label: '<%=rb.getString("CPEBianMa") %>',disabled: true,width: 230},
				{field: 'imsi', label: 'IMSI',sortable: 'custom',disabled: true,width: 150},
				{field: 'cpeName', label: '<%=rb.getString("CPEName") %>',sortable: 'custom',disabled: true,width: 150},
				{field: 'macAddress', label: '<%=rb.getString("CPEMacAddress") %>',width: 130},
				{field: 'directMode', label: '<%=rb.getString("MoShi") %>',sortable: false,disabled: true,width: 120},
				{field: 'PCI', label: 'PCI',sortable: 'custom',width: 60},
				{field: 'state', label: '<%=rb.getString("SASZhuangTai") %>',sortable: 'custom',disabled: true,width: 140},
				{field: 'cpiState', label: '<%=rb.getString("CPIZhuangTai") %>',sortable: 'custom',disabled: true,width: 120},
				{field: 'category', label: '<%=rb.getString("SASLeiXing") %>',width: 80},
				{field: 'cbsdId', label: '<%=rb.getString("CBSDID") %>',width: 180},
				{field: 'channelType', label: '<%=rb.getString("XinDaoLeiXing") %>',width: 180},
				{field: 'grants', label: '<%=rb.getString("ShouQuan") %>',width: 200},
				{field: 'maxEirp', label: '<%=rb.getString("SASEIRP") %>',width: 180},
				{field: 'frequency', label: '<%=rb.getString("PinLvHeZi") %>',width: 150},
				{field: 'grantExpireTime', label: '<%=rb.getString("grantGuoQiShiJian")%>',width: 150},
				{field: 'hostName', label: '<%=rb.getString("HostName") %>',sortable: 'custom',width: 150},
				{field: 'lgwMac', label: '<%=rb.getString("LGWMACDiZhi") %>',width: 180},
				{field: 'lgwIp', label: '<%=rb.getString("LGWIPDiZhi") %>',width: 180}
			],
			cpeSortColumns: [
				{field: 'serialNumber', label: '<%=rb.getString("CPEBianMa") %>',disabled: true,width: 230},
				{field: 'imsi', label: 'IMSI',sortable: true,disabled: true,width: 150},
				{field: 'cpeName', label: '<%=rb.getString("CPEName") %>',sortable: 'custom',disabled: true,width: 150},
				{field: 'macAddress', label: '<%=rb.getString("CPEMacAddress") %>',width: 130},
				{field: 'directMode', label: '<%=rb.getString("MoShi") %>',sortable: false,disabled: true,width: 120},
				{field: 'PCI', label: 'PCI',sortable: true,width: 60},
				{field: 'state', label: '<%=rb.getString("SASZhuangTai") %>',sortable: true,disabled: true,width: 140},
				{field: 'cpiState', label: '<%=rb.getString("CPIZhuangTai") %>',sortable: true,disabled: true,width: 120},
				{field: 'category', label: '<%=rb.getString("SASLeiXing") %>',width: 80},
				{field: 'cbsdId', label: '<%=rb.getString("CBSDID") %>',width: 180},
				{field: 'channelType', label: '<%=rb.getString("XinDaoLeiXing") %>',width: 180},
				{field: 'grants', label: '<%=rb.getString("ShouQuan") %>',width: 200},
				{field: 'maxEirp', label: '<%=rb.getString("SASEIRP") %>',width: 180},
				{field: 'frequency', label: '<%=rb.getString("PinLvHeZi") %>',width: 150},
				{field: 'grantExpireTime', label: '<%=rb.getString("grantGuoQiShiJian")%>',width: 150},
				{field: 'hostName', label: '<%=rb.getString("HostName") %>',sortable: true,width: 150},
				{field: 'lgwMac', label: '<%=rb.getString("LGWMACDiZhi") %>',width: 180},
				{field: 'lgwIp', label: '<%=rb.getString("LGWIPDiZhi") %>',width: 180}
			],
			cpeOrigionSort: [],

			showWindowInfo:false,
			dialogTitle:'',
			dialogUrl:'',
			windowWidth:'500px',
			listShow:false,
			isNoData:false,
			activeName: ['eNB','gNB','CPE'].includes(netType)? netType : 'eNB',
			queryForm:{
				opStatus:'',
				cbsdType:''
			},
			params_sasEnb:{
				timeZone: timeZone,
				like_fields:'serialNumber,hostName,cellIp',
				deviceType:'eNB',
				search_text:'',
				serialNumber:'',
				hostName:'',
				cbsdId:'',
				opStatus:'',
				cbsdType:''
			},
			params_sasGnb:{
				timeZone: timeZone,
				like_fields:'serialNumber,hostName,cellIp',
				deviceType:'gNB',
				search_text:'',
				serialNumber:'',
				hostName:'',
				cbsdId:'',
				opStatus:'',
				cbsdType:''
			},
			params_sasCpe:{
				timeZone: timeZone,
				like_fields:'imsi,cpeName,serial_number',
				deviceType:'CPE',
				search_text:'',
				serial_number: '',
				imsi:'',
				cpeName:'',
				macAddress:'',
				lgw_ip: '',
				lgw_mac: '',
				cbsdType:''
			},
			cpeUrl:'',
			enbUrl:'${ctx}/cell/SAS/getMonitorGridData.action',
			gnbUrl:'${ctx}/cell/SAS/getMonitorGridData.action',
			
			settingsEnbForm:{ // 设置SAS开关
				sasAble:'0',
				sasProvider:'',
				sasUserId:''
			},
			selectSasProvider:[
				{value:'0',label:'Federated Wireless'},
				{value:'2',label:'Amdocs'},
				{value:'3',label:'CommScope'},
				{value:'4',label:'Google'},
			],
			settingsInfo:{ // 设置详情页面的参数
				userId: '',
				callSign: ''
			},
			settingsRules:{
				userId:[
					{required:true,message:'<%=rb.getString("QingShuRu")%>User ID',trigger:'blur'},
				]
			},
			selectionData:[],
			slideUrl:'',
			slideTitle:'',
			slideTitleView:'',
			slideHeader:'',
			slideFooter:'',
			slidePosition:'',
			slideHeight:'',
			slideWidth:'',
			height:'100%',
			width:'100%',
			modal:false,
			tableRule:{
				'eNB':'enbTable',
				'gNB':'gnbTable',
				'CPE':'cpeTable',
			},
			isabled:false,
			enbSortSas:'',
			enbOrderSas:'',
			gnbSortSas:'',
			gnbOrderSas:'',
			cpeSortSas:'',
			cpeOrderSas:'',
		
			tsslideUrl:'',
			tsslideHeader:'',
			tsslideFooter:'',
			tsslidePosition:'',
			tsslideHeight:'',
			tsslideWidth:'',
			sasEnableCount:'0',
			totalCount:'0',
			grantSuspendedCount:'0',
			authorizedCount:'0',
			lastRequestedTime:'',
			enbTableData:[],
			grantsData:{
				grantList:[],
				temGrantList:[],
				backupGrantList:[],
			},
			grantsPopoverLoading:false,
			sasCertStatus:false,
			sasImportCertForm:{
				errorFlag:false,
				uploadFileURL:'',
				errorMessage:'Only .P12/.PEM are supported',
				fileName:'',
				certFile:'',
				password:'',
				cpiid:'',
				cpiname:'',
				p12:'true',
				keyData:''
			},
			importSasCertshow:'false',
			sasImportCertFormRules:{
				cpiid: [
					{required:true,message:'<%=rb.getString("BuNengWeiKong")%>',trigger:'blur'}
				],
				cpiname: [
					{required:true,message:'<%=rb.getString("BuNengWeiKong")%>',trigger:'blur'}
				],
				password:[
					{ validator:passwordValidator}
				]
			},
			enbBulkSelectShow:false,
			gnbBulkSelectShow:false,
			cpeBulkSelectShow:false,
			enb_search_text:'',
			gnb_search_text:'',
			cpe_search_text:'',
			placeholderText:'<%=rb.getString("QingShuRu")%>',
			opStatusList:[
				{value:'',label:'<%=rb.getString("QuanBu")%>'},
				{value:'1',label:'<%=rb.getString("JiHuo")%>'},
				{value:'0',label:'<%=rb.getString("QuJiHuo")%>'}
			],
			cbsdTypeList:[
				{value:'',label:'<%=rb.getString("QuanBu")%>'},
				{value:'1',label:'virtual'},
				{value:'0',label:'normal'}
			],
			rightBoxTitle:'',
			rightBoxType:'',

			sasImportSelectFlag: false,        //标识是否选择了文件 
			sasImportTypeFlag: true,
			sasImportFileName: '',
			sasImportForm: {
				userid: '',
				excelFile:'',
				uploadFileURL: '${ctx}/cell/SAS/importInstallParam.action',
			},
			sasImportFormRules: {
				userid: [
					{ validator: useridValidator },
				],
			},
		}		
	},
	computed: {
		limitBatch(){
			return batchOperation ? '' : 1;
		},
		showColums() {
			var vm = this,
				columns = vm.columns,
				props = columns.map(function(col){
					return col.field;
				});

			props = props.filter(function(code){
				return vm.showProps.includes(code);
			});

			return props;
		},
		gnbShowColums() {
			var vm = this,
				columns = vm.gnbColumns,
				props = columns.map(function(col){
					return col.field;
				});

			props = props.filter(function(code){
				return vm.gnbShowProps.includes(code);
			});

			return props;
		},
		cpeShowColums() {
			var vm = this,
				columns = vm.cpeColumns,
				props = columns.map(function(col){
					return col.field;
				});

			props = props.filter(function(code){
				return vm.cpeShowProps.includes(code);
			});

			return props;
		},
		dragOptions() {

			return {
				animation: 200,
				group: 'description',
				disabled: false,
				ghostClass: 'ghost'
			};
		},
		dragAll() {
			var vm = this;
			return vm.dragCol.length == vm.columns.length;
		},
		gnbDragAll() {
			var vm = this;
			return vm.gnbDragCol.length == vm.gnbColumns.length;
		},
		cpeDragAll() {
			var vm = this;
			return vm.cpeDragCol.length == vm.cpeColumns.length;
		},
		statisticsParam(){			
			return this.activeName
		},
		sasCertKeyValue(){
			return sessionStorage.getItem('keyData')
		},
        netType() {
            var vm = this,
                neType = sysMain.headType,
                codes = {
                    enb: 'eNB',
                    cpe: 'CPE',
                    gnb: 'gNB'
                };

            return codes[neType];
        },
		optBtnShow() {
			return writableMap['CODE_ADVANCE_SAS'] == true;
		},
	},
	mounted(){
		this.init();
		var vm = this;
		if(window.updateSasStatusTimer) clearInterval(window.updateSasStatusTimer);
		window.updateSasStatusTimer = setInterval(function(){
			var sasMonitorPageCtn = $("#sasMonitorPage");			
			if(!sasMonitorPageCtn.length) {
				clearInterval(window.updateSasStatusTimer);
				return;
			}
			var ctner = document.querySelector('#monitor_ctner'),
				visible = isVisible(ctner),
	    		isCovered = isOverlapped(ctner);
	    	
	    	if(visible && !isCovered) {
				vm.getCount(vm.statisticsParam);
	    	}
		},6000);
		clearInterval(refreshTable);
		refreshTable = setInterval(function(){
			var sasMonitorPageCtn = $("#sasMonitorPage");			
			if(!sasMonitorPageCtn.length) {
				clearInterval(window.updateSasStatusTimer);
				return;
			}

			if(vm.lastRequestedTime == '' || (vm.lastRequestedTime && vm.compareTime(vm.lastRequestedTime))){
				vm.refreshTableData();
				vm.lastRequestedTime = new Date();
			}
		},10000);
		
		eventBus.$off('close-dialog').$on('close-dialog',this.cancelViewSlide); 
		eventBus.$off('close-dialogs').$on('close-dialogs',this.cancelImport); 
		eventBus.$off('goProcedure').$on('goProcedure', this.goProperties);
	},
	watch:{
		sasCertKeyValue:{
			immediate:true,
			handler(val){
				if(val){
					this.sasCertStatus = true;
				}else{
					this.sasCertStatus = false;
				}
			},
			deep:true
		},
        netType(type) {
            var vm = this,
                types = ['eNB','gNB','CPE'];

            if(types.includes(type)) {
				vm.activeName = type;

				vm.$nextTick(function(){
					vm.tabsChange();
				})
			}
        }
	},
	methods:{
		// 模糊搜索
		query(){
			var vm = this,
				type = vm.activeName;
			if(type == 'eNB'){
				vm.params_sasEnb.search_text = this.enb_search_text;
			}else if(type == 'gNB'){
				vm.params_sasGnb.search_text = this.gnb_search_text;
			}else{
				vm.params_sasCpe.search_text = this.cpe_search_text;
			}
			
		},
		// 搜索域聚焦事件
		queryInputFocus(){
			var vm = this,
				type = vm.activeName;
			if(type == 'eNB' || type == 'gNB'){
				vm.placeholderText = '<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("HostName")%> / IP';
			}else{
				vm.placeholderText = 'IMSI / <%=rb.getString("CPEName")%> / <%=rb.getString("SheBeiWeiYiBiaoZhi") %>';
			}
			
		},
		// 搜索域失焦事件
		queryInputBlur(){
			var vm = this;
			vm.placeholderText = '<%=rb.getString("QingShuRu")%>';
		},
		// 高级查询 确定事件
		advanceQuery(){
			var vm = this,
			type = vm.activeName;
			if(type == 'eNB'){
				Object.assign(vm.params_sasEnb, vm.queryForm);
			}else if(type == 'gNB'){
				Object.assign(vm.params_sasGnb, vm.queryForm);
			}else{
				Object.assign(vm.params_sasCpe, vm.queryForm);
			}
		},
		// 清除筛选
		clearFilterClick(){
			var vm = this,
				type = vm.activeName,
				params = {
					opStatus:'',
					cbsdType:''
				};
			Object.assign(vm.queryForm, params);
			if(type == 'eNB'){
				Object.assign(vm.params_sasEnb, params);
			}else if(type == 'gNB'){
				Object.assign(vm.params_sasGnb, params);
			}else{
				Object.assign(vm.params_sasCpe, params);
			}
			document.body.click();
		},
		// 打开已选弹窗
		openBulkSelectTable(type){
			var vm = this,
				type = vm.activeName;
			if(type == 'eNB'){
				vm.enbBulkSelectShow = true
			}else if(type == 'gNB'){
				vm.gnbBulkSelectShow = true
			}else{
				vm.cpeBulkSelectShow = true
			}
			
		},
		// 关闭已选弹窗
		closeBulkSelectTable(type){
			var vm = this,
				type = vm.activeName;
			if(type == 'eNB'){
				vm.enbBulkSelectShow = false
			}else if(type == 'gNB'){
				vm.gnbBulkSelectShow = false
			}else{
				vm.cpeBulkSelectShow = false
			}
		},
		// 设备已选表格 清空事件
		clearBulkSelected(){
			var vm = this,
				type = vm.activeName;
			if(type == 'eNB'){
				vm.$refs["enbTable"].clearSelection();
			}else if(type == 'gNB'){
				vm.$refs["gnbTable"].clearSelection();
			}else{
				vm.$refs["cpeTable"].clearSelection();
			}
			
		},
		// 设备已选表格 单个删除事件
		delBulkSelected(rows){
			var vm = this,
				type = vm.activeName,
				codes={
					'eNB':'enbTable',
					'gNB':'gnbTable',
					'CPE':'cpeTable'
				},
				tabs = codes[type],
				rowKey = 'small_cell_code';
			vm.selectionData = vm.selectionData.filter((items)=>{
				return items[rowKey] != rows[rowKey]
			});
			var selection = this.$refs[tabs].$refs.ctableInner.store.states.selection,
				irow= selection.filter((items)=>{
					return items[rowKey] == rows[rowKey]
				})[0];
			vm.$refs[tabs].toggleRowSelection(irow,false);
			var idx = vm.$refs[tabs].ckList.indexOf(rows[rowKey])
			vm.$refs[tabs].ckList.splice(idx,1);
		},
		dragAllChange(val) {
			var vm = this,
				fields = vm.columns.map(function(item){
					return item.field;
				}),
				filters = vm.columns.filter(function(item){
					return item.disabled;
				}).map(function(item){
					return item.field;
				});
			
			vm.dragCol = val?fields:filters;
		},
		gnbDragAllChange(val) {
			var vm = this,
				fields = vm.gnbColumns.map(function(item){
					return item.field;
				}),
				filters = vm.gnbColumns.filter(function(item){
					return item.disabled;
				}).map(function(item){
					return item.field;
				});
			
			vm.gnbDragCol = val?fields:filters;
		},
		cpeDragAllChange(val) {
			var vm = this,
				fields = vm.cpeColumns.map(function(item){
					return item.field;
				}),
				filters = vm.cpeColumns.filter(function(item){
					return item.disabled;
				}).map(function(item){
					return item.field;
				});
			
			vm.cpeDragCol = val?fields:filters;
		},
		init(){
			var vm = this,		
				activeName = this.$root.activeName,
				params ={
					operatorCode:operator_code
				};
			axios.post('${ctx}/cell/SAS/getSasBillingDeviceEnable.action',stringify(params)).then(function(response){
				var data = response.data
				if(data.sasSetting == 'enable'){ 
					vm.isabled = false
				}else{
					vm.isabled = true
				}
				if(data.sasFunc == 'enable'){
					vm.listShow = true
					vm.isNoData = false
				}else{
					vm.listShow = false
					vm.isNoData = true
				}
			})
			$('.el-loading-mask').css('display','none')
			vm.getCount(vm.statisticsParam);

		},
		refreshTableData(){
			var vm = this,
				strNum = Math.random().toString();
			if(vm.activeName == 'eNB'){
				vm.enbUrl='${ctx}/cell/SAS/getMonitorGridData.action?randomCode='+strNum; 
			}else if(vm.activeName == 'gNB'){
				vm.gnbUrl='${ctx}/cell/SAS/getMonitorGridData.action?randomCode='+strNum; 
			}else{
				vm.cpeUrl='${ctx}/cell/SAS/getMonitorGridData.action?randomCode='+strNum; 
			}
		},
		// 时间比较  判断传入时间是否距当前时间3分钟
		compareTime(time){
			var vm = this;

			if(new Date().getTime() - time.getTime() >1000*60*3){
				return true
			}else{
				return false
			}
		},
		// 表格加载成功
		tableLoadSuccess(data){
			var vm = this;
			vm.lastRequestedTime = '';
		},
		// 弹窗关闭
		closeDialog(){
			var vm = this,
				params={
					userId: '',
					callSign: ''
				};
			vm.showWindowInfo = false;
			Object.assign(vm.settingsInfo,params);
			vm.dialogUrl = ''
		},
		// 弹窗打开成功传递参数  
		openDialogSuc(){
			eventBus.$emit('open-dialog');
		},
	    //如果是辅助小区，增加 carrierType字段，carrierType字段值为'scell'，表格操作项及SAS开关禁止点击 
	    isDisabled(row,index){
			if(row.carrierType == 'scell' || row.cbsdType =='virtual'){
				return false
			}else{
				return true
			}
		},
		isDisabledCpe(row,index){
			if(row.cbsdType =='virtual'){
				return false
			}else{
				return true
			}
		},
		/**
		 *  install params
		*/
	    goProperties(row){ 
			var vm = this;
			var activeName = '',
				sn = row.serialNumber,
				isVirtual = false;
			
			var curRow = Object.keys(row);
			if(curRow.includes('deviceType')){
				activeName = row.deviceType;
				vm.activeName = row.deviceType;
			}else{
				activeName = vm.$root.activeName
			}
			if(row.cbsdType == 'virtual'){
				isVirtual = true;
			}
			if(activeName == 'eNB' || activeName == 'gNB'){
				vm.$root.slideUrl = '${ctx}/cell/SAS/toInstallParamPage.action?serialNumber='+sn+'&deviceType='+activeName;
			}else{
				vm.$root.slideUrl = '${ctx}/cell/SAS/toSasCpeSetting.action';
			}
			
			vm.$root.slideTitleView = '<%=rb.getString("SheZhi")%>';
			vm.$root.slidePosition = 'top';
			vm.$root.slideHeight = '100%';
			vm.$root.slideFooter = false;
			vm.$root.slideHeader = false;
			vm.$root.$refs.slideView.showSlide(function(){
				eventBus.$emit('open-dialog','edit',activeName,row,sn,isVirtual);
			});
	    },
		// sas开关改变事件
	    sasEnabelChange(row) {
			var vm = this;
			
			var confirmMsg='', message='',url='${ctx}/cell/SAS/cellModifySASSwitch.action';
			var params = {
					sasEnable : row.sasEnable == 'on' ? 'off' : 'on',
					deviceCode : row.small_cell_code,
					serialNum:row.serialNumber,
					deviceType: vm.activeName
				}
			if(row.sasEnable === 'off'){
				confirmMsg = '<%=rb.getString("PiLiangKaiSAS")%>';
				message='<%=rb.getString("CaoZuoMingCheng")%>:<%=rb.getString("DaKaiSasKaiGuan")%>  <%=rb.getString("SheBeiMingCheng")%>'+ row.serialNumber
			}else{
				confirmMsg = '<%=rb.getString("LiXianSheBeiGuanBiSASTiShi")%>';
				message='<%=rb.getString("CaoZuoMingCheng")%>:<%=rb.getString("GuanBiSasKaiGuan")%>  <%=rb.getString("SheBeiMingCheng")%>'+ row.serialNumber
			}
			vm.$confirm(confirmMsg,'<%=rb.getString("QueRen")%>',{
				customClass:'warningConfirm',
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancelButtonText:'<%=rb.getString("QuXiao")%>',
				type:'warning',
				closeOnClickModal:false
			}).then(() => {
				vm.$refs[row.serialNumber].activeIconClass= 'el-icon-loading';
				axios.post(url,stringify(params)).then(function(response){
					let data = response.data
					if(data.success){
						vm.$notify({
							title:'<%=rb.getString("ChengGong")%>',
							message:message,
							type:'success',
							duration:10000,
						})
						vm.$refs[vm.tableRule[vm.activeName]].refresh();
					}else{
						vm.$notify({
							title:data.message,
							message:message,
							type:'error',
							duration:10000,
						})
					}
					vm.$refs[row.serialNumber].activeIconClass= '';
				}).catch(function(error){
					
				})
			}).catch()
			event.stopPropagation();
		},
	    // 导出文件
		exportCSV(type){ 
			var vm = this,params={},exportUrl='';
			var activeName = vm.$root.activeName;
			if(activeName == 'eNB'){
				params={
					search_text:vm.params_sasEnb.search_text,
					serialNumber:vm.params_sasEnb.serialNumber,
					hostName:vm.params_sasEnb.hostName,
					cbsdId:vm.params_sasEnb.cbsdId,
					opStatus:vm.params_sasEnb.opStatus,
					cbsdType:vm.params_sasEnb.cbsdType,
					deviceType:activeName,
					timeZone: timeZone,
					order:vm.enbOrderSas,
					sort:vm.enbSortSas,
					like_fields: 'serialNumber,hostName',
				}
		    }else if(activeName == 'gNB'){
				params={
					search_text:vm.params_sasGnb.search_text,
					serialNumber:vm.params_sasGnb.serialNumber,
					hostName:vm.params_sasGnb.hostName,
					cbsdId:vm.params_sasGnb.cbsdId,
					opStatus:vm.params_sasGnb.opStatus,
					cbsdType:vm.params_sasGnb.cbsdType,
					deviceType:activeName,
					timeZone: timeZone,
					order:vm.gnbOrderSas,
					sort:vm.gnbSortSas,
					like_fields: 'serialNumber,hostName',
				}
		    }else if(activeName == 'CPE'){
		    	params={
					search_text:vm.params_sasCpe.search_text,
					imsi:vm.params_sasCpe.imsi,
					cpeName:vm.params_sasCpe.cpeName,
					macAddress:vm.params_sasCpe.macAddress,
					cbsdType:vm.params_sasCpe.cbsdType,
					deviceType:activeName,
					timeZone: timeZone,
					order:vm.cpeOrderSas,
					sort:vm.cpeSortSas,
					serial_number: vm.params_sasCpe.serial_number,
					lgw_ip: vm.params_sasCpe.lgw_ip,
					lgw_mac: vm.params_sasCpe.lgw_mac,
					like_fields: 'imsi,cpeName,serial_number',
				}
		    }
			
			if (type == "google"){
				exportUrl ="${ctx}/cell/SAS/exportGoogleSasTemplate.action";
			}else if(type == "templateData"){
				exportUrl ="${ctx}/cell/SAS/exportToVirtualCbsdTemplateFormat.action";
			}else{
				exportUrl ="${ctx}/cell/SAS/exportMonitorGridData.action";
			}
			
			exportByForm(exportUrl,params);
		},
		// 日志
		sasLogs(row){
			var vm = this;
			vm.$root.slideUrl = "${ctx}/cell/SAS/toSasMainLog.action";
			vm.$root.slideTitleView = '';
			vm.$root.slidePosition = 'top';
			vm.$root.slideHeight = '100%';
			vm.$root.slideWidth = '100%';
			vm.$root.slideFooter = false;
			vm.$root.slideHeader = false;
			vm.$root.$refs.slideView.showSlide(function(){
				eventBus.$emit('sasLog-init',row,'single');
			});
		},
		// 右上角全部日志
		sasMainLog(){
			var vm = this;
			vm.$root.slideUrl = "${ctx}/cell/SAS/toSasMainLog.action";
			vm.$root.slideTitleView = '';
			vm.$root.slidePosition = 'top';
			vm.$root.slideHeight = '100%';
			vm.$root.slideWidth = '100%';
			vm.$root.slideFooter = false;
			vm.$root.slideHeader = false;
			vm.$root.$refs.slideView.showSlide(function(){
				eventBus.$emit('sasLog-init','','all');
			});
		},
		// 设置
		sasSetting(){
			var vm = this;
			axios.post('${ctx}/cell/SAS/getSasSettings.action').then(function(response){
				var data = response.data;
				vm.settingsEnbForm.sasAble = data.sasEnable;
				vm.settingsEnbForm.sasProvider = data.sasProvider;
				var providers=[];
				var urls = data.sasURL;
				if(urls && urls.length){
					var sasUrlList = {};
					urls.map((item,index) => {
						sasUrlList[item.id] = item.url;
						providers.push({
							value: item.id,
							label: item.providerName,
							sasUserId:item.sasUserId,
						})
					})
				}
				providers.map((item)=>{
					if(item.value == vm.settingsEnbForm.sasProvider){
						vm.settingsEnbForm.sasUserId = item.sasUserId 
					}
				})
				vm.selectSasProvider = providers;
				
			})
			vm.importSasCertshow = 'true';
			vm.rightBoxType = 'settings';
			vm.rightBoxTitle = '<%=rb.getString("SheZhi")%>';
		},
		setInstall(){ // 设置Install  params
			var vm= this;

			if(vm.selectionData.length <= 0)return
			vm.showWindowInfo = true;
			vm.dialogTitle = '<%=rb.getString("SheZhi")%>';
			vm.windowWidth = '360px';
		},
		submitSASSetting(){//sas设置提交事件
			var vm = this, url='${ctx}/cell/SAS/saveSasSettings.action',params={};
			params.sasEnable = vm.settingsEnbForm.sasAble;
			params.sasProvider = vm.settingsEnbForm.sasProvider;
			params.sasUserId = vm.settingsEnbForm.sasUserId;
			
			vm.$refs.settingsEnbForm.validate((r)=>{
				if(r){
					axios.post(url,stringify(params)).then(function(response){
						var data = response.data;
						if(data.success){
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success',
							});
							vm.rightBoxClose();
						}else{
							vm.$message({
								message: data.message,
								type:'error',
							});
						}
					})
				}
			})
		},
		// 批量操作 ———— sas 开关
		batchCbrsas(status){
			var vm = this,
				serialNumberList=[],
				codeList=[],
				confirmMsg='',
				message='',
				url='${ctx}/cell/SAS/updateSasEnableBatch.action',
				params = {
					sasEnable:status,
					deviceCode:'',
					serialNum:'',
					deviceType:vm.activeName
				};
			if(vm.selectionData.length <= 0)return
			vm.selectionData.map(function(item,idx){
				serialNumberList.push(item.serialNumber);
				codeList.push(item.small_cell_code);
			});
			params.serialNum = serialNumberList.join(',');
			params.deviceCode = codeList.join(',');
			if(status === 'on'){
				confirmMsg = '<%=rb.getString("PiLiangKaiSAS")%>'
				message='<%=rb.getString("CaoZuoMingCheng")%>:<%=rb.getString("DaKaiSasKaiGuan")%>  <%=rb.getString("SheBeiMingCheng")%>'+ params.serialNum
			}else{
				confirmMsg = '<%=rb.getString("LiXianSheBeiGuanBiSASTiShi")%>'
				message='<%=rb.getString("CaoZuoMingCheng")%>:<%=rb.getString("GuanBiSasKaiGuan")%>  <%=rb.getString("SheBeiMingCheng")%>'+ params.serialNum
			}
	    	vm.$confirm(confirmMsg,'<%=rb.getString("QueRen")%>',{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		closeOnClickModal:false
	    	}).then(() => {
	    		axios.post(url,stringify(params)).then(function(response){
					let data = response.data
					if(data.success){
						vm.$notify({
							title:'<%=rb.getString("ChengGong")%>',
							message:message,
							type:'success',
							duration:10000,
							customClass:"msgNotify"
						})
						vm.cancelBatchOpt()
						vm.$refs[vm.tableRule[vm.activeName]].refresh();
					}else{
						vm.$notify({
							title:data.message,
							message:message,
							type:'error',
							duration:10000,
							customClass:"msgNotify"
						})
					}
		    	}).catch(function(error){
		    		
		    	})
	    	}).catch()
		},
		// 批量操作 ———— sas 设置
		sasBatchParams(){
			var vm = this, 
			serialNumberList=[],
			codeList=[],
			url='${ctx}/cell/SAS/updateInstallParamBatch.action',
			params={
				userId:vm.settingsInfo.userId,
				callSign:vm.settingsInfo.callSign,
				deviceType:vm.activeName,
				serialNum:'',
				deviceCode:''
			};
			
			vm.selectionData.map(function(item,idx){
				serialNumberList.push(item.serialNumber);
				codeList.push(item.small_cell_code);
			});
			params.serialNum = serialNumberList.join(',');
			params.deviceCode = codeList.join(',');
			vm.$refs.settingsForm.validate((r)=>{
				if(r){
					axios.post(url,stringify(params)).then(function(res){
						var data= res.data
						if(data.success){
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success',
							});
							vm.closeDialog();
							vm.$refs[vm.tableRule[vm.activeName]].clearSelection();
							vm.$refs[vm.tableRule[vm.activeName]].refresh();
						}else{
							vm.$message({
								message: data.message,
								type:'error',
							});
						}
					})
				}
			})
		},
	    cancelBatchOpt(){ // 取消批量操作
	    	var vm = this;
	    	vm.delAllSelectedRecord();
	    },
		/**
		 *  查看操作
		 * @param row:当前数据
		*/
	    cancelViewSlide(){ // 二级页面关闭
	    	var vm = this;
	    	vm.$refs.slideView.hide();
			vm.$refs[vm.tableRule[vm.activeName]].refresh();		
	    },
		cancelImport(){ // 关闭导入页面
			var vm = this;
			
			vm.$refs.tslide.hide();
			vm.$refs[vm.tableRule[vm.activeName]].refresh();		
		},
	
		/**
		 * 多选响应
		 * @param selection:选择的数据
		*/
	    batchSelect(selection){
	    	var vm = this;

	    	vm.selectionData = selection
		},
		delAllSelectedRecord(){ // 清除当前tab的所有设备选择记录
			var vm = this;
			
			vm.$refs[vm.tableRule[vm.activeName]].clearSelection();	
		},
	    tabClick(tab){ // 列表的点击
	    	var vm = this;
	    },
		tabsChange(){ // 列表的点击
	    	var vm = this,
				type = vm.activeName,
				params = {
					opStatus:'',
					cbsdType:'',
				};
			vm.cancelViewSlide();
			//tab 切换时，导入窗口默认为关闭状态
			vm.$refs.tslide.hide();
			vm.getCount(vm.statisticsParam);
			vm.cancelBatchOpt();
			vm.selectionData = [];

			Object.assign(vm.queryForm, params);
			if(type == 'eNB'){
				Object.assign(vm.params_sasEnb, params);
			}else if(type == 'gNB'){
				Object.assign(vm.params_sasGnb, params);
			}else{
				Object.assign(vm.params_sasCpe, params);
			}
			document.body.click();
		},
	    // 统计数据
	    getCount(){
	    	var vm = this;
	    	var params={					
					deviceType:vm.statisticsParam				
				}
			axios.post('${ctx}/cell/SAS/queryStatistics.action',stringify(params)).then(function(response){
				var data = response.data
				if(data){
					vm.sasEnableCount = data.sasEnableCount;
					vm.totalCount = data.totalCount;
					vm.grantSuspendedCount = data.grantSuspendedCount;
					vm.authorizedCount = data.authorizedCount;
				}
			})
		},
	    //排序点击  --- eNB
		sortChangeEnb(data){ 
			var vm =this,
				order={
					ascending:'asc',
					descending:'desc'
				};
			vm.enbSortSas = data.prop;
			vm.enbOrderSas = order[data.order];		
		},
		//排序点击  --- gNB
		sortChangeGnb(data){ 
			var vm =this,
				order={
					ascending:'asc',
					descending:'desc'
				};
			vm.gnbSortSas = data.prop;
			vm.gnbOrderSas = order[data.order];		
		},
		//排序点击  --- CPE 
		sortChangeCPE(data){ 
			var vm =this,
				order={
					ascending:'asc',
					descending:'desc'
				};
			vm.cpeSortSas = data.prop;
			vm.cpeOrderSas = order[data.order];		
		},
		// 导入
		sasImportClick(){
			var vm = this;
			var activeName = this.$root.activeName;
			
			vm.importSasCertshow = 'true';
			vm.rightBoxType = 'sasImport';
			vm.rightBoxTitle = '<%=rb.getString("DaoRu")%> CBSDs';

			// vm.$root.tsslideUrl = "${ctx}/cell/SAS/toImportInstallParam.action";
			// vm.$root.slideTitleView = '<%=rb.getString("DaoRu")%> CBSDs';
			// vm.$root.tsslidePosition = '';
			// vm.$root.tsslideHeight = '440px';
			// vm.$root.tsslideWidth = '740px';
			// vm.$root.modal = false;
			// vm.$refs.tslide.showSlide(function(){
			// 	if(activeName == 'eNB'){
			// 		eventBus.$emit('hander-rows','eNB')
			// 	}else{
			// 		eventBus.$emit('hander-rows','CPE')
			// 	}				
			// });	
		},
		// 表格操作项删除
		deleteVirtual(row,code){
	    	var vm = this,
	    		url ='${ctx}/cell/SAS/deleteVirtualCbsd.action',
	    		confirmMsg = '<%=rb.getString("QueDingShanChuXuNiCBSD")%>',
	    		params = {
					serialNumber:row.serialNumber,
					deviceType:vm.activeName
	    		};	    		
	    	vm.$confirm(confirmMsg,'<%=rb.getString("QueRen")%>',{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		closeOnClickModal:false
	    	}).then(() => {
	    		axios.post(url,stringify(params)).then(function(response){
					let data = response.data
					if(data.success){
						vm.$message({
							message: '<%=rb.getString("ChengGong")%>',
							type:'success',
						});
						vm.$refs[vm.tableRule[vm.activeName]].refresh();
					}else{
						vm.$message({
							message: data.message,
							type:'error',
						});
					}
		    	}).catch(function(error){
		    		
		    	})
	    	}).catch()
	    },
		// 提供商改变
		sasProviderChange(val){
			var vm = this;
			vm.selectSasProvider.map((item)=>{
				if(item.value == val){
					vm.settingsEnbForm.sasUserId = item.sasUserId 
				}
			})
		},

		columnSetting() {
			$("#enb_sas_column_setting").slideDown();
		},
		gnbColumnSetting() {
			$("#gnb_sas_column_setting").slideDown();
		},
		cpeColumnSetting() {
			$("#cpe_sas_column_setting").slideDown();
		},
		enbColumnConfig() {
			var vm = this,
				url = '${ctx}/system/column/setting/insert.action',
				sortCol = [];

			vm.sortColumns.map(function(item){
				sortCol.push(item.field);
			});

			var params = {
					pageName: '3',
					showColumn: vm.dragCol.join(','),
					sortColumn: sortCol.join(',')	
				};

			axios.post(url, params).then(function(res){
				vm.columns = [];
				vm.sortColumns.map(function(item){
					vm.columns.push(Object.assign({}, item));
				});

				vm.showProps = [];
				vm.dragCol.map(function(code){
					vm.showProps.push(code);
				});

				vm.close();
			});
		},
		gnbColumnConfig() {
			var vm = this,
				url = '${ctx}/system/column/setting/insert.action',
				sortCol = [];

			vm.gnbSortColumns.map(function(item){
				sortCol.push(item.field);
			});

			var params = {
					pageName: '7',
					showColumn: vm.gnbDragCol.join(','),
					sortColumn: sortCol.join(',')	
				};

			axios.post(url, params).then(function(res){
				vm.gnbColumns = [];
				vm.gnbSortColumns.map(function(item){
					vm.gnbColumns.push(Object.assign({}, item));
				});

				vm.gnbShowProps = [];
				vm.gnbDragCol.map(function(code){
					vm.gnbShowProps.push(code);
				});

				vm.close();
			});
		},
		cpeColumnConfig() {
			var vm = this,
				url = '${ctx}/system/column/setting/insert.action',
				sortCol = [];

			vm.cpeSortColumns.map(function(item){
				sortCol.push(item.field);
			});

			var params = {
					pageName: '4',
					showColumn: vm.cpeDragCol.join(','),
					sortColumn: sortCol.join(',')	
				};

			axios.post(url, params).then(function(res){
				vm.cpeColumns = [];
				vm.cpeSortColumns.map(function(item){
					vm.cpeColumns.push(Object.assign({}, item));
				});

				vm.cpeShowProps = [];
				vm.cpeDragCol.map(function(code){
					vm.cpeShowProps.push(code);
				});

				vm.close();
			});
		},
		enbCancelConfig() {
			var vm = this;

			vm.dragCol = [];
			vm.showProps.map(function(code){
				vm.dragCol.push(code);
			});

			vm.close();
		},
		gnbCancelConfig() {
			var vm = this;

			vm.gnbDragCol = [];
			vm.gnbShowProps.map(function(code){
				vm.gnbDragCol.push(code);
			});

			vm.close();
		},
		cpeCancelConfig() {
			var vm = this;

			vm.cpeDragCol = [];
			vm.cpeShowProps.map(function(code){
				vm.cpeDragCol.push(code);
			});

			vm.close();
		},
		close() {
			$("#enb_sas_column_setting").slideUp(500);
			$("#gnb_sas_column_setting").slideUp(500);
			$("#cpe_sas_column_setting").slideUp(500);
			
		},
		grantsPopoverShow(row){ // 授权信息Popover 打开事件
			var vm = this,
				params={
					grantList:[],
					temGrantList:[],
					backupGrantList:[]
				};
			Object.assign(vm.grantsData,params);
			vm.grantsPopoverLoading = true;
			axios.post('${ctx}/cell/SAS/getSasGrants.action',stringify({serialNumber:row.serialNumber})).then(function(response){
				let data = response.data
				if(data){
					if(data.grant){
						var grantList = data.grant.split(',');
						grantList.map((item)=>{
							vm.grantsData.grantList.push(vm.strTrim(item))
						})
					}
					if(data.temGrant){
						var temGrantList = data.temGrant.split(',');
						temGrantList.map((item)=>{
							vm.grantsData.temGrantList.push(vm.strTrim(item))
						})
					}
					if(data.backupGrant){
						var backupGrantList = data.backupGrant.split(',');
						backupGrantList.map((item)=>{
							vm.grantsData.backupGrantList.push(vm.strTrim(item))
						})
					}
					vm.grantsPopoverLoading = false;
				}
			}).catch(function(error){})

		},
		grantsPopoverHide(){ // 激活信息Popover 关闭事件
			var vm = this,
				params={
					grantList:[],
					temGrantList:[],
					backupGrantList:[]
				};
			Object.assign(vm.grantsData,params);
		},
		// 导入证书文件
		sasCertImport(){
			var vm = this,
				params = {
					cpiid:'',
					cpiname:''
				};
			vm.importSasCertshow = 'true';
			vm.rightBoxType = 'sasCert';
			vm.rightBoxTitle = 'Verify CPI Identity';
			if(vm.sasCertStatus){
				params.cpiid = sessionStorage.getItem('sasCpiId') ? sessionStorage.getItem('sasCpiId') : '';
				params.cpiname = sessionStorage.getItem('sasCpiName') ? sessionStorage.getItem('sasCpiName') : '';
				Object.assign(vm.sasImportCertForm,params);
			}
		},
		// 导入证书文件关闭
		rightBoxClose(){
			var vm = this,
				type = vm.rightBoxType,
				params={
					errorFlag:false,
					uploadFileURL:'',
					fileName:'',
					password:'',
					cpiid:'',
					cpiname:'',
					p12:'true',
					keyData:''
				};
			vm.importSasCertshow = 'false';
			if(type == 'sasCert'){
				Object.assign(vm.sasImportCertForm,params);
			}else if(type == 'settings'){
				vm.$refs[vm.tableRule[vm.activeName]].refresh();
				var settingsEnbForm ={ // 设置SAS开关
					sasAble:'0',
					sasProvider:'',
					sasUserId:''
				}
				Object.assign(vm.settingsEnbForm,settingsEnbForm);
			}else if(type == 'sasImport'){
				vm.closeSasImportFileSelect();
			}
			
		},
		 /* PrivateKey 导入监听
		* file:文件
		*fileList：文件列表
		*/
		fileChangePrivateKey(file, fileList) {
			var vm = this;
			var errorFlag = file.name.substr(file.name.lastIndexOf(".")) === '.pem' || file.name.substr(file.name.lastIndexOf(".")) === '.p12'
			vm.sasImportCertForm.errorFlag = !errorFlag;
			if (errorFlag) {
				vm.sasImportCertForm.certFile = file.raw;
				vm.sasImportCertForm.fileName = file.name;
			} else {
				vm.sasImportCertForm.fileName = '';
			}
		},
		checkFile(res, file) {    //发送请求，校验device文件内容 
			var vm = this;
			if (res.success) {
				if (res.suc_count > 0) {
					vm.$message({
						type: 'success',
						message: '<%=rb.getString("ChengGong")%>'
					});
				} else {
					vm.$message({
						type: 'warning',
						message: '<%=rb.getString("ShiBai")%>'
					});
				}
			} else {
				vm.$message({
					type: 'error',
					message: res.msg
				});
			}
			//修改已选择文件状态  
			var fileList = vm.$refs.uploadCert.uploadFiles;
			fileList.forEach(function (file) {
				file.status = 'ready';
			})
		},
		//PrivateKey 导入文件按钮
		fileSelectPrivateKey() { 
			var vm = this;
			vm.$refs.uploadCert.clearFiles();
			vm.$refs['file_up'].click();
		},
		p12CheckChange(val){
			var vm =this;
			if(val == 'false'){
				vm.sasImportCertForm.password = ''
			}
		},
		// sas 证书导入确定
		importSasCertSubmit(){
			var vm = this;
			 if(vm.sasImportCertForm.fileName == ''){
				vm.sasImportCertForm.errorFlag = true;
				return
			}
			vm.$refs.sasImportCertForm.validate((valid) => {
				if (valid){

					var fileName = vm.sasImportCertForm.fileName;
					var fileType = fileName.substr(fileName.lastIndexOf(".")) === '.p12' ? 'P12' : 'PEM';
					if(fileType == 'P12'){
						if(vm.sasImportCertForm.p12 == 'false'){
							vm.$message({
								message: 'CPI Certificate password is incorrect',
								type:'error',
							});
							return
						}
						vm.parsingP12Cert(vm.sasImportCertForm.certFile);
					}else{
						vm.parsingPemCert(vm.sasImportCertForm.certFile);
					}
					setTimeout(()=>{
						if(vm.sasImportCertForm.keyData == ''){
							vm.$message({
								message: 'CPI Signature Data was not generated successfully',
								type:'error',
							});
							return
						}
						vm.$message({
							message: '<%=rb.getString("ChengGong")%>',
							type:'success',
						})
						vm.sasCertStatus = true;
						sessionStorage.setItem('keyData',vm.sasImportCertForm.keyData);
						sessionStorage.setItem('sasCpiId',vm.sasImportCertForm.cpiid);
						sessionStorage.setItem('sasCpiName',vm.sasImportCertForm.cpiname);
						sessionStorage.setItem('userCode',user_code);
						vm.rightBoxClose();
					},200)
				} else {
					return false;
				}
			})
		},
		// 解析P12证书
		parsingP12Cert(file){
			var vm = this;
			var reader = new FileReader();
			reader.onload = function(e){
				var content = e.target.result;
				var pkcs12der = vm.arrBufferToString(content);
				var pkcs12B64 = forge.util.encode64(pkcs12der);

				var pkcs12ders = forge.util.decode64(pkcs12B64);
				try {
					var p12Asn1 = forge.asn1.fromDer(pkcs12ders);
				} catch (error) {
					vm.$message({
						message: 'Invalid CPI Certificate file',
						type:'error',
					});
					return
				}
				
				try {
					var p12 = forge.pkcs12.pkcs12FromAsn1(p12Asn1,vm.sasImportCertForm.password);
				} catch (error) {
					vm.$message({
						message: 'CPI Certificate password is incorrect',
						type:'error',
					});
					return
				}

				var privateKey;
				for(var sci = 0; sci < p12.safeContents.length; sci++){
					var safeContents = p12.safeContents[sci];
					for(var sbi = 0; sbi < safeContents.safeBags.length; sbi++){
						var safeBag = safeContents.safeBags[sbi];

						if(safeBag.type === forge.pki.oids.pkcs8ShroudedKeyBag){
							privateKey = safeBag.key
						}
					}
				};
				
				var pkcs8 = vm.privateKeyToPkcs8(privateKey);
				vm.sasImportCertForm.keyData = pkcs8;
			}
			reader.readAsArrayBuffer(file);
		},
		// 解析Pem证书
		parsingPemCert(file){
			var vm = this;
			var reader = new FileReader();
			reader.onload = function(e){
				var content = e.target.result;
				var pemder = vm.arrBufferToString(content);
				try {
					var privateKey = forge.pki.privateKeyFromPem(pemder);
				} catch (error) {
					vm.$message({
						message: 'Invalid CPI Certificate file',
						type:'error',
					});
					return
				}
				var rsaPrivateKey = forge.pki.privateKeyToAsn1(privateKey);
				var privateKeyInfo = forge.pki.wrapRsaPrivateKey(rsaPrivateKey);
				var privateKeyInfoDer = forge.asn1.toDer(privateKeyInfo).getBytes();
				var p8b64 = forge.util.encode64(privateKeyInfoDer)

				vm.sasImportCertForm.keyData = p8b64;
			}
			reader.readAsArrayBuffer(file);
		},
		// 转换为PKCS8
		privateKeyToPkcs8(privateKey){
			var vm = this;
			var rsaPrivateKey = forge.pki.privateKeyToAsn1(privateKey);
			var privateKeyInfo = forge.pki.wrapRsaPrivateKey(rsaPrivateKey);
			var privateKeyInfoDer = forge.asn1.toDer(privateKeyInfo).getBytes();
			var p8b64 = forge.util.encode64(privateKeyInfoDer)
			return p8b64
		},
		// 数组存储转换成字符串
		arrBufferToString(buffer){
			var binary = '';
			var bytes = new Uint8Array(buffer);
			var len = bytes.byteLength;
			for(var i = 0; i<len; i++){
				binary += String.fromCharCode(bytes[i]);
			}
			return binary
		},
		// 字符串转换为数据存储
		stingToArrayBuffer(data){
			var arrBuff = new ArrayBuffer(data.length);
			var writer = new Uint8Array(arrBuff);
			for(var i = 0,len = data.length; i<len;i++ ){
				writer[i] = data.charCodeAt(i);
			}
			return arrBuff
		},
		//取消 SAS执行定时注册功能
		closeTimingExecution(row){
			var vm = this,
				params={
					dualCarrierType:row.carrierType ? row.carrierType : '',
					serialNumber:row.serialNumber,
					deviceType:vm.activeName.toLowerCase(),
					action:'0',
					cancelSchedule:true
				};
			axios.post("${ctx}/cell/SAS/operate.action",stringify(params)).then(function(response){
				var data = response.data;
				if(data["success"]){
					vm.$message({
						message: '<%=rb.getString("ChengGong")%>',
						type:'success',
					})
				}else{
					vm.$message({
						message:data.message,
						type:'error',
					})
				}
				
			})
		},
		sasStateFmt(val){
			var codeList={
				'0':'Unregistered',
				'1':'Registered',
				'3':'Granted',
				'4':'Grant Suspended',
				'5':'Authorized',
				'6':'Transmission',
			}
			return codeList[val]
		},
		 /*sas导入确定*/
		sasImportSubmit() {
			var vm = this,
				params={
					token:omctoken,
					userId:vm.sasImportForm.userid,
					excelFile:vm.sasImportForm.excelFile,
					deviceType:vm.activeName
				};
			params.cpiId = sessionStorage.getItem('sasCpiId') ? sessionStorage.getItem('sasCpiId') : '';
			params.cpiName = sessionStorage.getItem('sasCpiName') ? sessionStorage.getItem('sasCpiName') : '';
			params.keyData = sessionStorage.getItem('keyData') ? sessionStorage.getItem('keyData') : '';
			vm.$refs.sasImportForm.validate((valid) => {
				if (valid){
					if (!vm.sasImportFileName) {
						vm.sasImportSelectFlag = true
						return
					}
					vm.sasImportUploadFiles(vm.sasImportForm.uploadFileURL, params)
				} else {
					return false;
				}
			})
		},
		 /*
		* 导入函数
		* url：当前的修改或者添加url
		* params： 所有的from参数
		*/
		sasImportUploadFiles(url, params, cb) {
			var vm = this,
				xhr = new XMLHttpRequest(),
				fmd = new FormData();
			if (params) {
				for (var key in params) {
					fmd.append(key, params[key]);
				}
			}
			xhr.onreadystatechange = function () {
				if (this.readyState == 4 && xhr.status == 200) {
					var data = JSON.parse(xhr.responseText);
					if (cb && typeof cb == 'function') cb(data);
				}
			}
			xhr.open('post', url);
			xhr.send(fmd);
			xhr.onreadystatechange = function () {
				if (xhr.response) {
					let str = xhr.response;
					str = JSON.parse(str)
					if (str.success) {
						vm.$message.success('<%=rb.getString("ChengGong")%>')
						vm.rightBoxClose();
					} else {
						vm.$message.error(str.message)
					}
				}
			}
		},
		/**
		*选择文件后，校验格式，并赋值页面显示 
		*@param file：文件名称
		*/
		sasImportFileChange(file, fileList) {
			var vm = this;
			vm.sasImportSelectFlag = false;
			var typeFlag = file.name.substr(file.name.lastIndexOf(".")) === '.xls' || file.name.substr(file.name.lastIndexOf(".")) === '.xlsx'
			vm.sasImportTypeFlag = typeFlag;
			if (typeFlag) {
				vm.sasImportForm.excelFile = file.raw;
				vm.sasImportFileName = file.name;
			} else {
				vm.sasImportFileName = '';
			}
		},
		
		sasImportCheckFile(res, file) {    //发送请求，校验device文件内容 
			var vm = this;
			if (res.success) {
				if (res.suc_count > 0) {
					vm.$message({
						type: 'success',
						message: '<%=rb.getString("ChengGong")%>'
					});
				} else {
					vm.$message({
						type: 'warning',
						message: '<%=rb.getString("ShiBai")%>'
					});
				}
			} else {
				vm.$message({
					type: 'error',
					message: res.msg
				});
			}
			//修改已选择文件状态  
			var fileList = vm.$refs.uploadSas.uploadFiles;
			fileList.forEach(function (file) {
				file.status = 'ready';
			})
		},
		closeSasImportFileSelect() { // 关闭文件选择 暂时没有用到 后期维护代码 去掉uploadFiles会用到
			var vm = this,
				params = {
					userid: '',
					excelFile:'',
					uploadFileURL: '${ctx}/cell/SAS/importInstallParam.action',
				};
			Object.assign(vm.sasImportForm,params)
			vm.sasImportFileName = '';
			vm.sasImportTypeFlag = true;
			vm.sasImportSelectFlagc = false;
			vm.$refs.uploadSas.clearFiles();
		},
		sasImportFileSelect() {  /// 导入文件按钮
			var vm = this;
			vm.$refs.uploadSas.clearFiles();
			vm.$refs['file_up'].click();
		},
		exportSasImportTemp(){
			var vm = this;
			exportByForm("${ctx}/cell/SAS/downloadImportInstallParamTemplate.action",{deviceType:vm.activeName})
				
		},
		// 去除首尾空格
		strTrim(str) {
			return str.replace(/(^\s*)|(\s*$)/g, "");
		},
	},
	created() {
		var vm = this;

		axios.post('${ctx}/system/column/setting/load/3').then(function(res){
			var data = res.data.data,
				sortCodes = (data.sortColumn||'').split(','),
				sortList = vm.columns;

			(data.showColumn||'').split(',').map(function(code){
				if(!vm.showProps.includes(code)) {
					vm.showProps.push(code);
				}
				if(!vm.dragCol.includes(code)) {
					vm.dragCol.push(code);
				}
			});
			vm.origionSort = sortCodes;

			vm.columns = sortList.sort(function(n, m) {
				var idxn = sortCodes.indexOf(n.field)==-1?100:sortCodes.indexOf(n.field),
					idxm = sortCodes.indexOf(m.field)==-1?100:sortCodes.indexOf(m.field);

				return idxn - idxm;
			});
			vm.sortColumns = sortList.sort(function(n, m) {
				var idxn = sortCodes.indexOf(n.field)==-1?100:sortCodes.indexOf(n.field),
					idxm = sortCodes.indexOf(m.field)==-1?100:sortCodes.indexOf(m.field);

				return idxn - idxm;
			});
		});
		// 请求gnb的列显示数据
		axios.post('${ctx}/system/column/setting/load/7').then(function(res){
			var data = res.data.data,
				sortCodes = (data.sortColumn||'').split(','),
				sortList = vm.gnbColumns;

			(data.showColumn||'').split(',').map(function(code){
				if(!vm.gnbShowProps.includes(code)) {
					vm.gnbShowProps.push(code);
				}
				if(!vm.gnbDragCol.includes(code)) {
					vm.gnbDragCol.push(code);
				}
			});
			vm.gnbOrigionSort = sortCodes;

			vm.gnbColumns = sortList.sort(function(n, m) {
				var idxn = sortCodes.indexOf(n.field)==-1?100:sortCodes.indexOf(n.field),
					idxm = sortCodes.indexOf(m.field)==-1?100:sortCodes.indexOf(m.field);

				return idxn - idxm;
			});
			vm.gnbSortColumns = sortList.sort(function(n, m) {
				var idxn = sortCodes.indexOf(n.field)==-1?100:sortCodes.indexOf(n.field),
					idxm = sortCodes.indexOf(m.field)==-1?100:sortCodes.indexOf(m.field);

				return idxn - idxm;
			});
		});
		axios.post('${ctx}/system/column/setting/load/4').then(function(res){
			var data = res.data.data,
				sortCodes = (data.sortColumn||'').split(','),
				sortList = vm.cpeColumns;

			(data.showColumn||'').split(',').map(function(code){
				if(!vm.cpeShowProps.includes(code)) {
					vm.cpeShowProps.push(code);
				}
				if(!vm.cpeDragCol.includes(code)) {
					vm.cpeDragCol.push(code);
				}
			});
			vm.cpeOrigionSort = sortCodes;

			vm.cpeColumns = sortList.sort(function(n, m) {
				var idxn = sortCodes.indexOf(n.field)==-1?100:sortCodes.indexOf(n.field),
					idxm = sortCodes.indexOf(m.field)==-1?100:sortCodes.indexOf(m.field);

				return idxn - idxm;
			});
			vm.cpeSortColumns = sortList.sort(function(n, m) {
				var idxn = sortCodes.indexOf(n.field)==-1?100:sortCodes.indexOf(n.field),
					idxm = sortCodes.indexOf(m.field)==-1?100:sortCodes.indexOf(m.field);

				return idxn - idxm;
			});
		});
	}
})
</script>