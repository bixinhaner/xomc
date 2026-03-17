<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
#gnbFileContent .device_item{
	position:relative;
}
#gnbFileContent .device_item .el-ctable-toolbar {
	padding: 0 !important;
}
#gnbFileContent .el-icon-goback:before,.el-icon-status-upgrading:before{
	color:#fff;
	font-size:14px;
}
#gnbFileContent .advanceQuery{
	margin-left: 20px;
}

#gnbFileContent .list_item .el-tabs__item{
	font-size:14px;
}
#gnbFileContent .list_item .el-tabs--top{
	border:none;
}
#gnbFileContent .list_item{
	position:relative;
}
#gnbFileContent .list_type{
	position:absolute;
	left:20px;
	top:10px;
	z-index: 999;
}
#gnbFileContent .task_count{
	position:absolute;
	right:10px;
	top:15px;
}
#gnbFileContent .queryGroup{
	height:28px;
}

#gnbFileContent .file_item .queryGroup{
	margin-left:20px;
}
#gnbFileContent .file_item .el-ctable-toolbar {
	padding: 0 !important;
}
#gnbFileContent .queryInfo label{
	display:block;
	margin-bottom:5px;
	line-height:26px;
}
#gnbFileContent .queryInfo{
	display:inline-block;
}
#gnbFileContent .el-badge{
	position:relative;
}
#gnbFileContent .el-badge__content{
	position:absolute;
	top:6px;
	right:-4px;
	transform:translateY(-50%) translateX(100%);
	background-color:transparent;
	border-radius:10px;
	color:#fff;
	display:inline-block;
	font-size:10px;
	height:12px;
	line-height:11px;
	padding:0 6px;
	text-align:center;
	white-space:nowrap;
	cursor:default;
	border:1px solid transparent;
}
#gnbFileContent .el-icon-star-badge:before{
	color:#F3916C;
}
#gnbFileContent .el-date-editor .el-range__close-icon{
	line-height:20px;
}
#gnbFileContent .commonTabsTop .el-tabs__header {
	border: 1px solid #D5DCEC;
	border-bottom: 0;
	border-radius: 8px 8px 0 0;
	padding: 0 20px;
}
#gnbFileContent .list_item .el-tabs__header {
	border: 0;
}
#gnbFileContent .upgradeItemBoxCls{
	height: 100%;
	position: relative;
	display: flex;
}
#gnbFileContent .upgradeItemBoxCls >div{
	overflow: hidden;
}
#gnbFileContent .upgradeItemBoxCls .upgradeMainPageBox{
	position: relative;
	flex: 1;
}
#gnbFileContent .upgradeItemBoxCls .importFileBoxCls{
	flex: 0 1 360px;
	margin-left: 10px;
	position: relative;
	background-color: #FFFFFF;
	box-shadow: 0px 0px 10px 1px #E9EDF9;
	border-radius: 10px;
	border: 1px solid #E9EDF9;
	box-sizing: border-box;
}
#gnbFileContent .newTabs .el-ctable-toolbar{
	padding: 0px!important;
}
#gnbFileContent .el-radio.is-bordered,.addDeviceDialog .el-radio.is-bordered{
	height: 30px;
	padding: 7px 20px 0 10px;
}
#gnbFileContent .el-radio.is-bordered+.el-radio.is-bordered,.addDeviceDialog .el-radio.is-bordered+.el-radio.is-bordered{
	margin-left: 15px;
}
#gnbFileContent .el-tabs__header{
	border-bottom: 1px solid #E9E9E9 !important;
}
#gnbFileContent .rightOutBoxHeadCls{
	height: 50px;
	display: flex;
	align-items: center;
	font-weight: 600;
	font-size: 14px;
	justify-content: space-between;
	padding: 0px 20px;
	border-bottom: 1px solid #E9EDF9;
}
#gnbFileContent .rightItemMainBox{
	padding: 20px;
}
#gnbFileContent .rightItemMainBox .el-input,#gnbFileContent .rightItemMainBox .el-select{
	width: 100%;
}
#gnbFileContent .importFileBoxCls .footer{
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
#gnbFileContent .greyIcon::before{
	color: #7A7992;
	font-size: 14px;
}
#gnbFileContent .labelSlotCls > span{
	color: #999999;
	font-size: 12px;
	margin-left: 10px;
}
#gnbFileContent .el-radio-button:focus:not(.is-focus):not(.is-disabled){
	-webkit-box-shadow: none!important;
	box-shadow: none!important;
}
#gnbFileContent .importFileBoxCls .el-select .el-input.is-disabled .el-input__inner,#gnbFileContent .importFileBoxCls .el-select .el-input__inner{
    height: unset !important;
}
</style>
<!-- gNB升级页面 -->
<div class='panelDefault' id="gnbFileContent" style='border: none'>
	<div class="upgradeItemBoxCls">
		<div class="upgradeMainPageBox">
			<el-tabs v-model="activeName" style='height:calc(100% - 2px)' class='commonTabsTop'>
				
				<el-tab-pane label='<%=rb.getString("ShengJi")%>&<%=rb.getString("HuiTui")%>' name='upgrade' class='upgrade_item' style='overflow:auto'>
					<div style='flex:1;overflow:hidden; border: 1px solid #D5DCEC; border-radius: 0 0 8px 8px; border-top: 0;' class='device_item'>
						<el-ctable ref="upgrade_cell_table" id="upgrade_cell_table" row-key="small_cell_code" :url="deviceUrl" :height="height" :query-params="query_cell_params" pagination="true" @selection-change="selectCell">
							<el-table-column v-if="optBtnShow" type="selection" width="45" reserve-selection=true></el-table-column>
							<el-table-column prop="connection_status" width="50">
								<template slot-scope="scope">
									<div :class="{
										'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
										'':scope.row.have_connected==2,
										'conn_exc':scope.row.connection_status=='Exception',
										'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
								</template>
							</el-table-column>
							<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>' sortable min-width="200"></el-table-column>
							<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>' sortable min-width="300"></el-table-column>
							<el-table-column prop="PHYCELLID" label="<%=rb.getString("PCI2")%>" sortable min-width="120"></el-table-column>
							<el-table-column prop="cell_ip" label="<%=rb.getString("IPDiZhi")%>" sortable min-width="120"></el-table-column>
							<el-table-column prop='rollback_version' label='<%=rb.getString("HuiTuiBanBen")%>' sortable min-width="200"></el-table-column>
							<el-table-column prop='software_version' label='<%=rb.getString("RuanJianBanBen")%>' sortable min-width="200"></el-table-column>
							<el-table-column prop='product_type' label='<%=rb.getString("ChanPinLeiXingBiaoZhi")%>' min-width="120"></el-table-column>
							<el-table-column prop='module_type' label='<%=rb.getString("SheBeiXingHaoMing")%>' min-width="120"></el-table-column>
							<el-table-column prop='group_name' label='<%=rb.getString("SheBeiZu")%>' min-width="220" sortable></el-table-column>
							<template slot='toolbar'>
								<div class='toolbarHeadBtnBoxCls commonQuery' style="height:45px;">
									<div class="newIconBoxCls-bt CODE_GNB_UPGRADE_IMAGE hidden" style="right:56px;top:10px;" @click="addUpgradeTask" tip="<%=rb.getString("ShengJi")%>">
										<span class='el-icon el-icon-circle-upgrade'></span>
									</div>
									<div class="newIconBoxCls-bt CODE_GNB_ROLLBACK hidden" style="right:17px;top:10px;" @click="addRbTask" tip="<%=rb.getString("HuiTui")%>">
										<span class='el-icon el-icon-circle-restore'></span>
									</div>
									<el-query type="normal" @query="query" placeholder="<%=rb.getString("XiaoZhanBianMa")%>/<%=rb.getString("HostName")%>/<%=rb.getString("IPDiZhi")%>"></el-query>							
									<div class="tableHeadQueryBoxCls">
										<div v-for="(item,index) in advancedQueryItemList">
											<div v-if="item.type == 'checkbox' && item.isShow" style="margin-right:10px;">
												<el-popfilter
													:label='item.label'
													v-model="item.checkedItemList"
													:list="item.options"
													:visible.sync="item.isShow"
													@check-change="advanceQuery(item.type,item.value,item.checkedItemList)">
												</el-popfilter>
											</div>
											<div v-if="item.type == 'select' && item.isShow" style="margin-right:10px;">
												<el-popfilter
													type="single"
													:label='item.label'
													v-model="item.selectVal"
													:list="item.options"
													:visible.sync="item.isShow"
													@check-change="advanceQuery(item.type,item.value,item.selectVal)">
												</el-popfilter>
											</div>
										</div>
										<div class="advancedQueryItemBox"  style="background: #FFF;" @click="clearFilterClick">
											<%=rb.getString("QingKongShaiXuan")%>
										</div>
									</div>
								</div>
								
								<div style='height:43px;width:100%;background:#FFFFFF;line-height:43px; border-bottom: 1px solid #D5DCEC;'>
									<span style='font-weight:bold;margin-left:10px;margin-right:15px;'><%=rb.getString("ChangPinXingHao")%>:</span>
									<el-radio-group size="mini" v-model='productValue' @change="changeProduct" class='commonRadioButton2'>
										<el-radio-button v-for="item in productTypeList" :label="item.value">{{item.name}}</el-radio-button>									
									</el-radio-group>
								</div>							
							</template>
						</el-ctable>
					</div>
					<div style='flex:1;margin-top:10px;overflow:auto; border: 1px solid #D5DCEC; border-radius: 8px;' class='list_item'>
						<el-tabs v-model="list_name" style='height:100%' class='newTabs'>
							<el-tab-pane label="<%=rb.getString("RuanJianShengJi")%>" name="software" style="position:relative">
								<el-radio-group v-model="list_type" class="list_type commonRadioButton" @change="changeListType" size="mini">
									<el-radio-button label="task"><%=rb.getString("RenWuLieBiao")%></el-radio-button>
									<el-radio-button label="device"><%=rb.getString("BackupRestoreSheBeiLieBiao")%></el-radio-button>
								</el-radio-group>
								<el-ctable v-show="showTask" id="upgrade_task_table" ref="upgrade_task_table" time=6 :url="taskUrl" :height="height" :query-params="query_task_params" pagination="true" @load-success="loadSuccessTask">
									<el-table-column label='' width="30" class-name="no-text-tips">
										<template slot-scope="scope">
											<div class="el-icon el-icon-operation-more" @click="optClickTask(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
										</template>
									</el-table-column>
									<el-table-column prop='TASK_NAME' label='<%=rb.getString("RenWuMingCheng")%>' min-width="400"></el-table-column>
									<el-table-column prop='CREATE_USER' label='<%=rb.getString("CaoZuoRen")%>' min-width="100"></el-table-column>
									<el-table-column prop='CREATE_TIME' label='<%=rb.getString("CaoZuoShiJian")%>' min-width="200"></el-table-column>
									<el-table-column prop='VERSION' label='<%=rb.getString("BanBen")%>' min-width="200"></el-table-column>
									<el-table-column prop="TASK_STATUS" label='<%=rb.getString("ZhuangTai")%>' min-width="120" >
										<template slot-scope="scope">
											<div v-html="changePasswordTaskTableStatus(scope.row.TASK_STATUS)"></div>
										</template>
									</el-table-column>
									<el-table-column prop="TASK_PROGRESS" label='<%=rb.getString("JinDu")%>' min-width="100"></el-table-column>
									<el-table-column prop="TASK_RESULT" label='<%=rb.getString("JieGuo")%>' min-width="100" :formatter="resultFmt"></el-table-column>
									<el-table-column prop="START_TIME" label='<%=rb.getString("KaiShiShiJian")%>' min-width="180"></el-table-column>
									<el-table-column prop="END_TIME" label='<%=rb.getString("JieShuShiJian")%>' min-width="180"></el-table-column>
									<template slot="toolbar">
										<div class='toolbarHeadBtnBoxCls commonQuery' style="height:45px;">
											<el-query type="normal" style='margin-left:198px;' @query="queryTask" placeholder="<%=rb.getString("RenWuMingCheng")%>"></el-query>
											<el-date-picker style='margin-left: 20px;' 
												v-model="dateValue"
												type="datetimerange"
												value-format="yyyy-MM-dd HH:mm:ss"
												range-separator="——"  
												@change="dateChange"
												start-placeholder='<%=rb.getString("KaiShiShiJian")%>' 
												end-placeholder='<%=rb.getString("JieShuShiJian")%>'>
											</el-date-picker>
										</div>								
										<div class='task_count'>
											<p><span><i class='el-icon el-icon-status-waiting1'></i><%=rb.getString("DengDai")%></span><span>{{waitNum}}</span></p>
											<p><span><i class='el-icon el-icon-status-inProgress'></i><%=rb.getString("JinXingZhong")%></span><span>{{progressNum}}</span></p>
											<p><span><i class='el-icon el-icon-status-suspend'></i><%=rb.getString("ZanTing")%></span><span>{{suspendNum}}</span></p>
											<p><span><i class='el-icon el-icon-status-terminate'></i><%=rb.getString("YiJieShu")%></span><span>{{endNum}}</span></p>
										</div>
									</template>
								</el-ctable>
								<el-ctable v-show="!showTask" ref="upgrade_result_table" id="upgrade_result_table" time=6  :url="resultUrl" :height="height" :query-params="query_result_params" pagination="true" @load-success="loadsuccessResult"
								@selection-change="selectDevice">
									<!--<el-table-column type="selection" width="45" :selectable="choseDevice"></el-table-column>-->
									<el-table-column label='' width="30" class-name="no-text-tips" v-if="optBtnShow">
										<template slot-scope="scope">
											<div v-if="scope.row.result == '3'" class="el-icon el-icon-operation-restart" @click="restartTask('single',scope.row.taskId,scope.row.smallCellCode)" style="cursor: pointer;"></div>
										</template>
									</el-table-column>
									<el-table-column prop='serialNumber' label='<%=rb.getString("XiaoZhanBianMa")%>' min-width="200"></el-table-column>
									<el-table-column prop='cellName' label='<%=rb.getString("HostName")%>' min-width="200"></el-table-column>
									<el-table-column prop='taskName' label='<%=rb.getString("RenWuMingCheng")%>' min-width="300"></el-table-column>
									<el-table-column prop='originalVersion' label='<%=rb.getString("ChuShiBanBen")%>' min-width="200"></el-table-column>
									<el-table-column prop='upgradeVersion' label='<%=rb.getString("ShengJiBanBen")%>' min-width="200"></el-table-column>
									<el-table-column prop="status" label='<%=rb.getString("ZhuangTai")%>' min-width="120" >
										<template slot-scope="scope">
											<div v-html="resultTableStatus(scope.row.status)"></div>
										</template>
									</el-table-column>
									<el-table-column prop="result" label='<%=rb.getString("JieGuo")%>' min-width="100" :formatter="deviceResultFmt"></el-table-column>
									<el-table-column prop='failureReason' label='<%=rb.getString("ShiBaiYuanYin")%>' min-width="200"></el-table-column>
									<el-table-column prop='startTime' label='<%=rb.getString("KaiShiShiJian")%>' min-width="200"></el-table-column>
									<el-table-column prop='endTime' label='<%=rb.getString("JieShuShiJian")%>' min-width="200"></el-table-column>
									<template slot="toolbar">
										<div class='toolbarHeadBtnBoxCls' style='height:45px; margin-left: 220px; position: relative;'>
											<div class="newIconBoxCls-bt" style="right:20px;top:10px;" @click="exportResult" tip="<%=rb.getString("DaoChu")%>">
												<span class='el-icon el-icon-operation-export'></span>
											</div>
											<div class="queryGroup commonSearchWarp">
												<el-input class='pairgrid-query' v-model='query_result_form.searchText'  @keyup.enter.native="queryResult"
														placeholder="<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("HostName")%> / <%=rb.getString("RenWuMingCheng")%>"></el-input>
												<i @click='queryResult' class="el-icon el-icon-common-search ml10" ></i>
											</div>
											<div style='position: absolute;right:60px;top:15px;'>
												<p class='suc_count'><span><i class='el-icon el-icon-circle-success'></i><%=rb.getString("ChengGong")%></span><span>{{sucNum}}</span></p>
												<p class='fail_count'><span><i class='el-icon el-icon-circle-close'></i><%=rb.getString("ShiBai")%></span><span>{{failNum}}</span></p>
											</div>
										</div>								
									</template>
								</el-ctable>
							</el-tab-pane>
							<el-tab-pane label="<%=rb.getString("ShengJiHuiTuiRenWu")%>" name="rollback">
								<el-radio-group v-model="list_type_rb" class="list_type commonRadioButton" @change="changeListRb" size="mini">
									<el-radio-button label="task"><%=rb.getString("RenWuLieBiao")%></el-radio-button>
									<el-radio-button label="device"><%=rb.getString("BackupRestoreSheBeiLieBiao")%></el-radio-button>
								</el-radio-group>
								<el-ctable v-show="showTaskRb" ref="rb_task_table" id="rb_task_table" time=6 :url="taskUrlRb" :height="height" :query-params="query_task_params_rb" pagination="true" @load-success="loadSuccessTask">
									<el-table-column label='' width="30" class-name="no-text-tips">
										<template slot-scope="scope">
											<div class="el-icon el-icon-operation-more" @click="optClickTask(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
										</template>
									</el-table-column>
									<el-table-column prop='TASK_NAME' label='<%=rb.getString("RenWuMingCheng")%>' min-width="400"></el-table-column>
									<el-table-column prop='CREATE_USER' label='<%=rb.getString("CaoZuoRen")%>' min-width="300"></el-table-column>
									<el-table-column prop='CREATE_TIME' label='<%=rb.getString("CaoZuoShiJian")%>' min-width="200"></el-table-column>
									<el-table-column prop="TASK_STATUS" label='<%=rb.getString("ZhuangTai")%>' min-width="120" >
										<template slot-scope="scope">
											<div v-html="changePasswordTaskTableStatus(scope.row.TASK_STATUS)"></div>
										</template>
									</el-table-column>
									<el-table-column prop="TASK_PROGRESS" label='<%=rb.getString("JinDu")%>' min-width="100"></el-table-column>
									<el-table-column prop="TASK_RESULT" label='<%=rb.getString("JieGuo")%>' min-width="100" :formatter="resultFmt"></el-table-column>
									<el-table-column prop="START_TIME" label='<%=rb.getString("KaiShiShiJian")%>' min-width="180"></el-table-column>
									<el-table-column prop="END_TIME" label='<%=rb.getString("JieShuShiJian")%>' min-width="180"></el-table-column>
									<template slot="toolbar">
										<div class='toolbarHeadBtnBoxCls commonQuery' style="height:45px;">
											<el-query type="normal" style='margin-left:198px;' @query="queryTask" placeholder="<%=rb.getString("RenWuMingCheng")%>"></el-query>
											<el-date-picker style='margin-left: 20px;' 
												v-model="dateValueRb"
												type="datetimerange"
												value-format="yyyy-MM-dd HH:mm:ss"
												range-separator="——"  
												@change="dateChangeSoft"
												start-placeholder='<%=rb.getString("KaiShiShiJian")%>' 
												end-placeholder='<%=rb.getString("JieShuShiJian")%>'>
											</el-date-picker>
										</div>
										<div class='task_count'>
											<p><span><i class='el-icon el-icon-status-waiting1'></i><%=rb.getString("DengDai")%></span><span>{{waitNum}}</span></p>
											<p><span><i class='el-icon el-icon-status-inProgress'></i><%=rb.getString("JinXingZhong")%></span><span>{{progressNum}}</span></p>
											<p><span><i class='el-icon el-icon-status-suspend'></i><%=rb.getString("ZanTing")%></span><span>{{suspendNum}}</span></p>
											<p><span><i class='el-icon el-icon-status-terminate'></i><%=rb.getString("YiJieShu")%></span><span>{{endNum}}</span></p>
										</div>
									</template>
								</el-ctable>
								<el-ctable v-show="!showTaskRb" ref="rb_result_table" id="rb_result_table" time=6 :url="resultUrlRb" :height="height" :query-params="query_result_params_rb" pagination="true" @load-success="loadsuccessResult">
									<el-table-column prop='serialNumber' label='<%=rb.getString("XiaoZhanBianMa")%>' min-width="200"></el-table-column>
									<el-table-column prop='cellName' label='<%=rb.getString("HostName")%>' min-width="200"></el-table-column>
									<el-table-column prop='taskName' label='<%=rb.getString("RenWuMingCheng")%>' min-width="300"></el-table-column>
									<el-table-column prop='originalVersion' label='<%=rb.getString("ChuShiBanBen")%>' min-width="200"></el-table-column>
									<el-table-column prop="status" label='<%=rb.getString("ZhuangTai")%>' min-width="120" >
										<template slot-scope="scope">
											<div v-html="resultTableStatus(scope.row.status)"></div>
										</template>
									</el-table-column>
									<el-table-column prop="result" label='<%=rb.getString("JieGuo")%>' min-width="100" :formatter="deviceResultFmt"></el-table-column>
									<el-table-column prop='failureReason' label='<%=rb.getString("ShiBaiYuanYin")%>' min-width="200"></el-table-column>
									<el-table-column prop='startTime' label='<%=rb.getString("KaiShiShiJian")%>' min-width="200"></el-table-column>
									<el-table-column prop='endTime' label='<%=rb.getString("JieShuShiJian")%>' min-width="200"></el-table-column>
									<template slot="toolbar">
										<div class='toolbarHeadBtnBoxCls' style='height:45px; margin-left: 220px; position: relative;'>		
											<div class="newIconBoxCls-bt" style="right:20px;top:10px;" @click="exportResult" tip="<%=rb.getString("DaoChu")%>">
												<span class='el-icon el-icon-operation-export'></span>
											</div>
											<div class="queryGroup commonSearchWarp">
												<el-input class='pairgrid-query' v-model='query_result_form_rb.searchText'  @keyup.enter.native="queryResult"
														placeholder="<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("HostName")%> / <%=rb.getString("RenWuMingCheng")%>"></el-input>
												<i @click='queryResult' class="el-icon el-icon-common-search ml10" ></i>
											</div>
											<div style='position: absolute;right: 60px;top:15px;'>
												<p class='suc_count'><span><i class='el-icon el-icon-circle-success'></i><%=rb.getString("ChengGong")%></span><span>{{sucNum}}</span></p>
												<p class='fail_count'><span><i class='el-icon el-icon-circle-close'></i><%=rb.getString("ShiBai")%></span><span>{{failNum}}</span></p>
											</div>
										</div>
									</template>
								</el-ctable>
							</el-tab-pane>
							<el-cmenu ref="menu_task" :data="menus_task" @click="clickMenuTask"></el-cmenu>
						</el-tabs>
					</div>
					<el-bulk ref="deviceBulk" target="upgrade_result_table" :list="selectionDevice" row-key="serialNumber" show-prop="serialNumber"
								:message="{title:'<%=rb.getString("YiXuanSheBei")%>',subTitle:'<%=rb.getString("XiaoZhanBianMa")%>',clear:'<%=rb.getString("QingKong")%>',cancel:'<%=rb.getString("QuXiao")%>'}">
						<template slot="button">
							<a class="linkbutton linkbutton_trend" @click="restartTask('batch')"><span><%=rb.getString("ChongXinZhiXing")%></span></a>
						</template>
					</el-bulk>
				</el-tab-pane>
				<!-- 文件页面 -->
				<el-tab-pane label='<%=rb.getString("WenJian")%>' name='file' class='file_item' style='position:relative'>
					<el-ctable ref="file_table"  :url="file_url" :height="height" :query-params="query_file_params" pagination="true" style='border: 1px solid #D5DCEC;border-top:none;height: calc(100% - 2px); border-radius: 0 0 8px 8px;'>
						<el-table-column label='' width="30" class-name="operationColumn">
							<template slot-scope="scope">
								<div class="el-icon el-icon-operation-more" @click="optClickFile(scope.row,event)" v-clickoutside="handerClose" ></div>
							</template>
						</el-table-column>
						<el-table-column label='<%=rb.getString("WenJianMing")%>' min-width="300" prop="file_name"></el-table-column>
						<el-table-column label='<%=rb.getString("BanBen")%>' min-width="240"  prop="version" show-overflow-tooltip="true">
							<template slot-scope="scope">
								<div v-if="scope.row.recommend == '1'" class="el-badge">
									<span>{{scope.row.version}}</span>
									<span class='el-badge__content el-icon el-icon-star-badge'></span>
								</div>
								<div v-else>{{scope.row.version}}</div>
							</template>
						</el-table-column>
						<el-table-column label='<%=rb.getString("ChanPinLeiXingBiaoZhi")%>' min-width="250" prop="product"  v-if="showProductFile"></el-table-column>
						<el-table-column label='<%=rb.getString("WenJianDaXiao")%>' min-width="200" prop="size"></el-table-column>
						<el-table-column v-if="isCloudCore=='true'?true:false" label='<%=rb.getString("BanBenLeiXing")%>' min-width="200" prop="toWho" :formatter="towhoFmt"></el-table-column>
						<el-table-column label='<%=rb.getString("ShangChuanShiJian")%>' min-width="220" prop="upload_time"></el-table-column>
						<template slot="toolbar">
							<div class='toolbarHeadBtnBoxCls' style="height:45px;">
								<div class="newIconBoxCls-bt CODE_GNB_UPGRADE_FILE hidden" style="right:20px;top:10px;" @click="importFileClick" tip="<%=rb.getString("DaoRuWenJian")%>">
									<span class='el-icon el-icon-operation-import'></span>
								</div>
								<div class='queryGroup commonSearchWarp'>
									<el-input v-model='query_file_form.searchText' @keyup.enter.native="queryFile" class='pairgrid-query' placeholder='<%=rb.getString("BanBen")%>'></el-input>
									<i @click='queryFile' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
								</div>
							</div>
						</template>
					</el-ctable>
				</el-tab-pane>
				<el-cmenu ref="menu_file" :data="menus_file" @click="clickMenuFile"></el-cmenu>
			</el-tabs>
		</div>
		<div class="importFileBoxCls" v-show="importFileShow">
			<div class="rightOutBoxHeadCls">
				<span>{{rightOutBoxTitle}}</span>
				<span class="el-icon el-icon-close greyIcon" @click="rightBoxClose"></span>
			</div>
			<div class="rightItemMainBox">
				<el-form ref="importFileForm" :model="importFileForm" label-position="top" :rules="importFileFormRules" :hide-required-asterisk=true>
					<el-form-item prop="productValue" label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" v-if="file_type != 'ap'" key="productValue">
						<el-select v-model='importFileForm.productValue' :disabled="importViewFlag">	
							<el-option v-for='item in productTypeList' :label="item.name" :value="item.value"></el-option>
						</el-select>
					</el-form-item>
					<el-form-item label="<%=rb.getString("WenJianMing")%>" prop="fileName">
						<span slot="label" class="labelSlotCls">
							<%=rb.getString("WenJianMing")%>
							<span v-show="importFileType == 'add'" >( {{fileTypeTip}} )</span>
						</span>
						<el-upload 
							v-show="importFileType == 'add'" 
							:before-upload='beforeUpload'  
							:on-success='checkImportFile' 
							:on-change="importFileChange" 
							:show-file-list=false 
							ref="importFile" 
							:action="importFileForm.importFileUrl"
							:auto-upload="false">
							<el-input :readonly="true" :value="importFileForm.fileName">
								<a slot="append" class="el-icon el-icon-operation-import greyIcon" @click="importFileSelect"></a>
							</el-input>
							<a slot="trigger" ref="file_up"></a>
						</el-upload>
						<el-input v-show="importFileType != 'add'" v-model='importFileForm.fileName' :disabled="true"></el-input>
					</el-form-item>
					<el-form-item label="<%=rb.getString("BanBen")%>" prop="version">
						<el-input v-model='importFileForm.version' :disabled="importViewFlag"></el-input>
					</el-form-item>
					<el-form-item label='<%=rb.getString("TuiJian")%>' prop='recommend'>
						<el-select v-model='importFileForm.recommend' :disabled="importViewFlag">
							<el-option label='<%=rb.getString("Shi")%>' value='1'></el-option>
							<el-option label='<%=rb.getString("Fou")%>' value='0'></el-option>
						</el-select>
					</el-form-item>
					<el-form-item label="<%=rb.getString("MiaoShu")%>" prop='desc'>
						<el-input v-model='importFileForm.desc' type='textarea' :rows='2' :disabled="importViewFlag"></el-input>
					</el-form-item>
				</el-form>
			</div>
			<div class="footer" v-show="importFileType !== 'view'" >
				<div class="lnkbuttonGroup"  style="margin-left:20px;" >
					<el-button type="primary" @click="importFileSubmit"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="rightBoxClose"><%=rb.getString("QuXiao")%></el-button>
				</div>
			</div>
		</div>
	</div>
	<el-slide ref="slide" :url="slideUrl" :title="slideTitle" :footer="slideFooter" :header='slideHeader' :position="slidePosition" class='commonBorderSlide'
	 :height="slideHeight" :modal='slideModal'  :width="slideWidth" :subloading="slideSubmitLoading" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'"  @cancel='cancelSlide' @ok='saveSlide'>
	 </el-slide>
</div>
<script>
var gnbFileVue = new Vue({
	el:'#gnbFileContent',
	data(){
		var vm = this,
			versionValidate = function(rule,value,callback) {
				if(value) {
					if(value.length>45) {
						callback('<%=rb.getString("FanWei")%>:1-45 <%=rb.getString("ZiFuFuShu")%>');
					}else {
						callback();
					}
				}else {
					callback('<%=rb.getString("QingShuRuWenJianBanBen")%>');
				}
			},
			fileNameValidate = (rule,value,callback) => {
				var reg = vm.difFileObj['upgrade'].fileFmt;
				var message = vm.difFileObj['upgrade'].fileErrorMsg;

				if(value == ""){
					callback(new Error("<%=rb.getString("QingXianXuanZeWenJian")%>"))
				}else if(!vm.fileFormatMatch(value,reg)){
					callback(new Error(message))
				}else{
					var pathSplit = value.split(/\\/),
						filename = pathSplit[pathSplit.length - 1];
					
					if(filename.length>100) {
						callback("<%=rb.getString("WenJianMingBuNengChaoGuoYaoQiu")%>");
					}else {
						callback();
					}
				}
			};
		return{
			activeName : 'upgrade',
			deviceUrl:'',
			height:"100%",
			//productValue:'(FAP/\\w+BU2210)|^001$|^003$|^5G BBU-F1$',
			productValue:'',
			query_cell_params:{
				group_id:'',
				isGnb:1,
				like_fields: 'serial_number,host_name,cell_ip',
				serial_number:'',
				host_name:'',
				software_version:'',
				search_text:'',
				cell_ip:'',
				productValue:'',
				rollback_version:"",
				rd: ''
			},
			query_cell_form:{
				group_id:'',
				serial_number:'',
				host_name:'',
				software_version:'',
				search_text:'',
				cell_ip:'',
				rollback_version:""
			},
			list_name:'software',
			list_type:'task',
			seach_device_text:'',
			showTask:true,
			rollbackForm:{
				status:'active',
				exetime:''
			},
			setTimeEnable:true,
			pickerOptions:{
				disabledDate(time){
					return time.getTime()< Date.now()-8.64e7;
				}
			},
			showRollback:false,
			file_type:'upgrade',
			fileImportForm:{
				fileName:''
			},
			showFileImport:false,
			slideUrl:'',
			slideTitle:'',
			slideFooter:'',
			slideHeader:'',
			slidePosition:'',
			slideHeight:'',
			slideWidth:'',
            slideSubmitLoading:'',
			slideModal:'',
			url:'',
			taskUrl:'${ctx}/task/upgrade/getUpgradeTaskList.action',
			taskUrlRb:"",
			dateValue:[],
			query_task_params:{
				timeZone:timeZone,
				isGnb:1,
				taskName:'',
				startTime:'',
				endTime:'',
				searchText:'',
				listType:'upgrade',
				rd: ''
			},
			query_task_form:{
				taskName:'',
				startTime:'',
				endTime:'',
				searchText:''
			},
			list_type_rb:"task",
			showTaskRb:true,
			query_task_params_rb:{
				timeZone:timeZone,
				isGnb:1,
				taskName:'',
				startTime:'',
				endTime:'',
				searchText:'',
				listType:"rollback",
				rd: ''
			},
			query_task_form_rb:{
				taskName:'',
				startTime:'',
				endTime:'',
				searchText:''
			},
			dateValueRb:[],
			file_url:'${ctx}/cell/version/queryfileInfosList.action?file_type=0',
			query_file_params:{
				timeZone:timeZone,
				isGnb:1,
				searchText:'',
				isMix:true,
				rd: ''
			},
			query_file_form:{
				searchText:''
			},
			menus_file:[],
			rowDataFile:[],
			operType:'',
			menus_task:[],
			rowDataTask:[],
			cellData:[],
			waitNum:"0",
			progressNum:"0",
			suspendNum:"0",
			endNum:"0",
			sucNum:"0",
			failNum:"0",
			resultUrl:"${ctx}/task/upgrade/getUpgradeDeviceList/upgrade.action",
			query_result_params:{
				searchText:"",
				isGnb:1,
				timeZone:timeZone,
				rd: ''
			},
			query_result_form:{
				searchText:""
			},
			query_result_params_rb:{
				searchText:"",
				isGnb:1,
				timeZone:timeZone,
				rd: ''
			},
			query_result_form_rb:{
				searchText:""
			},
			resultUrlRb:"",
			showProductFile:true,
			selectionDevice:[],
			productTypeList:[
				//{value:'(FAP/\\w+BU2210)|^001$|^003$|^5G BBU-F1$',text:'BaiBNX'},
				//{value:'FAP/\\w*BSC\\w+',text:'BaiBNQ'},
			],
			productOptions: [
				//{value:'(FAP/w+BU2210)|^001$|^003$|^5G BBU-F1$',text:'BaiBNX'},
				//{value:'FAP/w*BSCw+',text:'BaiBNQ'},
			],
			advancedQueryItemList:[
				{
					type:'select',
					isShow:true,
					popoverShow:false,
					selectVal:'',
					label:'<%=rb.getString("HuiTuiBanBen") %>',
					options:[],
					value:'rollback_version',
				},
				{
					type:'select',
					isShow:true,
					popoverShow:false,
					selectVal:'',
					label:'<%=rb.getString("RuanJianBanBen") %>',
					options:[],
					value:'software_version',
				},
				{
					type:'select',
					isShow:true,
					popoverShow:false,
					selectVal:'',
					label:'<%=rb.getString("SheBeiZu") %>',
					options:[],
					value:'group_id',
				},
				
			],
			importFileShow:false,
			importFileType:'',
			importViewFlag:false,
			importFileForm:{
				productValue:'',
				file:'',
				fileName:'',
				version:'',
				recommend:'1',
				desc:'',
				to_who:'all',
			},
			importFileFormRules:{
				fileName:[
					{validator: fileNameValidate}
				],
				version:[
					{validator: versionValidate,trigger:'blur'}
				],
				productValue:[
					{required:true,message:'<%=rb.getString("ShuRuBiTianXiang")%>',trigger:'change'}
				],
			},
			fileTypeTip:'<%=rb.getString("ZhiZhiChiIMGHeEXTGeShi")%>',
			fileErrorData:'',
			difFileObj:{
				upgrade:{
					fileTip:'<%=rb.getString("ZhiChiIMGGeShi")%>',
					fileTipYD:'<%=rb.getString("ShengJiZhiChiGeShi")%>',
					fileFmt:'IMG,ext',
					fileFmtYD:'tar.gz',
					fileErrorMsg:'<%=rb.getString("ZhiChiIMGHeEXTGeShi")%>',
					fileErrorMsgYD:'<%=rb.getString("ShengJiZhiZhiChiWenJian")%>'
				}
			},
			multipartMaxFileSize:'${multipartMaxFileSize}'
		}
	},
	methods:{
		init(){
			var vm = this;
			
			axios.post("${ctx}/task/upgrade/getProductType.action?isGnb=1").then(function(res){
            	var data = res.data;
            	if(data.length > 0){
            		vm.productTypeList = data;
    		   		vm.productValue = data[0].value;
    		   		vm.$nextTick(function(){
    					vm.deviceUrl = '${ctx}/task/upgrade/queryCellInfos.action?forSelect=1';
    				});
    		   		vm.query_cell_params.productValue = data[0].value;
            	}else{
            		vm.$nextTick(function(){
    					vm.deviceUrl = '${ctx}/task/upgrade/queryCellInfos.action?forSelect=1';
    				});
            		vm.query_cell_params.productValue = '';
            	}
		   		
			});
			
			if(isJumpToPage){
				if(isJumpToPage.type == 'view'){
					vm.activeName = 'file';
					setTimeout(function(){
						isJumpToPage = '';
					},10)
				}else if(isJumpToPage.type == 'upgrade'){
					vm.addUpgradeTask();
				}
			}
			
			setTimeout(function(){
				vm.commonSelection();
			},500)
		},
		
		query(val){
			this.query_cell_params.search_text = val;
			this.query_cell_params.rd = Math.random();
		},
		advanceQuery(){
			this.query_cell_form.search_text = "";
			Object.assign(this.query_cell_params,this.query_cell_form);
		},
		resetQuery(){
			this.$refs.query_cell_form.resetFields();
		},
		setTime(){
			this.rollbackForm.exetime = formatDate(new Date(gloableTime));
			this.$refs.rollbackForm.validateField('exetime');
		},
		addUpgradeTask(){
			var vm = this;
			vm.slideHeader = true;
    	    vm.slideTitle = '<%=rb.getString("XinJianRenWu")%>';
    	    vm.slideUrl = "${ctx}/gnb/upgrade/toGnodeBTaskAddPage.action";
    	    vm.slideFooter = 'true';
    	    vm.slidePosition = 'top';
    	    vm.slideHeight = '100%';
    	    vm.slideWidth = '100%';
            vm.slideSubmitLoading = false;
    	    vm.operType = 'addTask';
    	    vm.$refs.slide.showSlide(function(){
    	    	vm.modal = false;
				eventBus.$emit('task-init','','addTask');
    	    });
		},
		cancelSlide(){
			if(this.operType == "addTask" ||　this.operType == "modifyTask"){
				eventBus.$emit('cancel-add-task');
			}else if(this.operType == "importFile"){
				eventBus.$emit('cancel-import');
			}else if(this.operType == "editFile"){
				eventBus.$emit('cancel-import');
			}else{
				this.$refs.slide.hide();
			}
		},
		saveSlide(){
			if(this.operType == "addTask" ||　this.operType == "modifyTask"){
				eventBus.$emit('add-task');
			}
			if(this.operType == "importFile"){
				eventBus.$emit('save-imort-file');
			}
			if(this.operType == "editFile"){
				eventBus.$emit('save-imort-file');
			}
		},
		commonSelection(){
			var vm = this;
			
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				isGnb : 1,
				selectType : 'deviceGroup'
			})).then(function(response){
				var data = response.data,
					groupOptionsList=[];

				data.map((item)=>{
					groupOptionsList.push({label:item.text,value:item.value})
				})
				vm.advancedQueryItemList.map((items)=>{
					if('group_id' == items.value){
						items.options = groupOptionsList
					}
				})
			}).catch(function(error){})
			
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				isGnb : 1,
				selectType : 'version'
			})).then(function(response){
				var data = response.data,
					versionOptionsList=[];

				data.map((item)=>{
					versionOptionsList.push({label:item.text,value:item.value})
				})
				vm.advancedQueryItemList.map((items)=>{
					if('software_version' == items.value){
						items.options = versionOptionsList
					}
				})
			}).catch(function(error){})
			
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				isGnb : 1,
				selectType : 'rollbackVersion'
			})).then(function(response){
				var data = response.data,
					rollbackVersionList=[];

				data.map((item)=>{
					rollbackVersionList.push({label:item.text,value:item.value})
				})
				vm.advancedQueryItemList.map((items)=>{
					if('rollback_version' == items.value){
						items.options = rollbackVersionList
					}
				})
			}).catch(function(error){})			
		},
		queryTask(val){
			if(this.list_name == "software"){
				this.query_task_params.searchText = val;
				this.query_task_params.rd = Math.random();
			}else if(this.list_name == "rollback"){
				this.query_task_params_rb.searchText = val;
				this.query_task_params_rb.rd = Math.random();
			}
		},
		advanceQueryTask(){
			var form;
			var params;
			var value;
			if(this.list_name == "software"){
				form = this.query_task_form;
				params = this.query_task_params;
				value = this.dateValue;
			}else if(this.list_name == "rollback"){
				form = this.query_task_form_rb;
				params = this.query_task_params_rb;
				value = this.dateValueRb;
			}
			form.searchText = "";
			if(value != null){
				form.startTime = value[0];
    			form.endTime = value[1];
			}
			Object.assign(params,form);
		},
		resetQueryTask(){
			var form;
			var params;
			var value;
			if(this.list_name == "software"){
				form = this.query_task_form;
				params = this.query_task_params;
				this.dateValue = '';
			}else if(this.list_name == "rollback"){
				form = this.query_task_form_rb;
				params = this.query_task_params_rb;
				this.dateValueRb = '';
			}
			form.taskName = '';
			form.startTime = '';
			form.endTime = '';
		},
		resultFmt(row,column,cellValue,index){//软件升级 任务结果fmt
	    	var resultObj = {
					"1" : '<%=rb.getString("ChengGong")%>',
					"2" : '<%=rb.getString("BuFenChengGong")%>',
					"3" : '<%=rb.getString("ShiBai")%>',
					"" : ""
			}
			return resultObj[cellValue];
	    },
	    deviceResultFmt(row,column,value,index) {
	    	if (value == "3") {
				return ShiBai;
			} else if (value == "1") {
				return ChengGong;
			} else if (value == "2") {
				return ZhongZhi;
			}else {
				return "";
			}
	    },
	    changeListType(val){
	    	this.showTask = val=='task'?true:false;
	    },
	    changeListRb(val){
	    	this.showTaskRb = val=='task'?true:false;
	    },
	    queryFile(){
	    	Object.assign(this.query_file_params,this.query_file_form);
	    	this.query_file_params.rd = Math.random();
	    },
	    importFileClick(){
	    	var vm = this;

			vm.importFileType = 'add';
			vm.importViewFlag = false;
			vm.$refs.importFileForm.resetFields();
			vm.importFileShow = true;
	    },
	    optClickTask(row,ev){
	    	var vm = this,
                status = row.TASK_STATUS,
                type = row.TYPE,
                codeList = {
                    'software': 'CODE_GNB_UPGRADE_IMAGE hidden',
                    'rollback': 'CODE_GNB_ROLLBACK hidden'
                },
                codeName = codeList[vm.list_name] || '';
            vm.rowDataTask = row;
	    	vm.menus_task= [
		          {label:'<%=rb.getString("KaiShi")%>',cls:codeName,code:'start'},
		          {label:'<%=rb.getString("ZanTing")%>',cls:codeName,code:'stop'},
		          {label:'<%=rb.getString("ZhongZhiRenWu")%>',cls:codeName,code:'terminate'},
		          {label:'<%=rb.getString("XinXi")%>',code:'view'},
		          {label:'<%=rb.getString("XiuGai")%>',cls:codeName,code:'edit'},
		          {label:'<%=rb.getString("ShanChu")%>',cls:codeName,code:'del'},
		    ]
	    	initTaskStatus(status,vm.menus_task);
	    	vm.$nextTick(function(){
	    		document.body.click();
   		    	vm.$refs.menu_task.show(ev);
	    	});
	    },
	    clickMenuTask(ev){
	    	var codes = {
   	    		view:this.viewTask,
   	    		start:this.activeTask,
   	    		stop:this.suspendTask,
   	    		terminate:this.terminateTask,
   	    		edit:this.editTask,
   	    		del:this.delTask,
    	    }
   	    	if(codes[ev.code]){
   	    		codes[ev.code](this.rowDataTask["TASK_ID"],this.rowDataTask["TYPE"])
   	    	}
	    },
	    activeTask(task_id,type){ // 开始
	    	var vm = this;
	    	axios.post('${ctx}/task/upgrade/activeTask.action',stringify({
	    		taskId : task_id,
	    		type : type
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
	    			if(vm.list_name == "software"){
	    				vm.$refs.upgrade_task_table.refresh()
	    			}else{
	    				vm.$refs.rb_task_table.refresh()
	    			}
	    		}else{
	    			vm.$message.error(data["message"])
	    		}
	    	})
	    },
	    suspendTask(task_id,type){ // 暂停
	    	var vm = this;
	    	axios.post('${ctx}/task/upgrade/suspendTask.action',stringify({
	    		taskId : task_id,
	    		type : type
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
	    			if(vm.list_name == "software"){
	    				vm.$refs.upgrade_task_table.refresh()
	    			}else{
	    				vm.$refs.rb_task_table.refresh()
	    			}
	    		}else{
	    			vm.$message.error(data["message"])
	    		}
	    	})
	    },
	    terminateTask(task_id,type){ // 终止任务
	    	var vm = this;
	    	axios.post('${ctx}/task/upgrade/terminateUpgradeTask.action',stringify({
	    		taskId : task_id,
	    		type : type
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
	    			if(vm.list_name == "software"){
	    				vm.$refs.upgrade_task_table.refresh()
	    			}else{
	    				vm.$refs.rb_task_table.refresh()
	    			}
	    		}else{
	    			vm.$message.error(data["message"])
	    		}
	    	})
	    },
        viewTask(task_id,type){ 
            var vm = this;
            vm.slideTitle = '<%=rb.getString("XinXi")%>';
            vm.operType = 'viewTask';
            vm.slideFooter = false;
            if(vm.list_name == "software"){
	    		vm.slideUrl = "${ctx}/gnb/upgrade/toGnodeBTaskAddPage.action";
	    	}else{
	    		vm.slideUrl = '${ctx}/gnb/upgrade/goAddRollbackTask.action';
	    	}
	    	vm.slideHeader = true;
    	    vm.slidePosition = 'top';
    	    vm.slideHeight = '100%';
    	    vm.slideWidth = '100%';
            vm.slideSubmitLoading = false;
    	    vm.$refs.slide.showSlide(function(){
    	    	vm.modal = false
				eventBus.$emit('task-init',task_id,vm.operType);
    	    });
	    },
	    editTask(task_id,type){ //  修改任务
	    	var vm = this;
	    	vm.slideTitle = '<%=rb.getString("XiuGai")%>';
            vm.operType = 'modifyTask';
            vm.slideFooter = true;
	    	if(vm.list_name == "software"){
	    		vm.slideUrl = "${ctx}/gnb/upgrade/toGnodeBTaskAddPage.action";
	    	}else{
	    		vm.slideUrl = '${ctx}/gnb/upgrade/goAddRollbackTask.action';
	    	}
	    	vm.slideHeader = true;
    	    vm.slidePosition = 'top';
    	    vm.slideHeight = '100%';
    	    vm.slideWidth = '100%';
            vm.slideSubmitLoading = false;
    	    vm.$refs.slide.showSlide(function(){
    	    	vm.modal = false
				eventBus.$emit('task-init',task_id,vm.operType);
    	    });
	    },
	    delTask(task_id,type){//软件升级  删除任务
	    	var vm = this;
	    	this.$confirm('<%=rb.getString("QueRenShanChuRenWu")%>',QueRen,{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		closeOnClickModal:false
	    	}).then(() => {
	    		axios.post('${ctx}/task/upgrade/delUpgradeTask.action',stringify({
		    		taskId:task_id,
		    		type:type
		    	})).then(function(response){
		    		var data = response.data;
		    		if(data["success"]){
		    			if(vm.list_name == "software"){
		    				vm.$refs.upgrade_task_table.refresh()
		    			}else{
		    				vm.$refs.rb_task_table.refresh()
		    			}
		    			vm.$message({
			    			type:'success',
			    			message:'<%=rb.getString("ChengGong")%>'
			    		})
		    		}else{
		    			vm.$message.error(data["message"])
		    		}
		    	}).catch(function(error){
		    		
		    	})
	    	}).catch()
	    },
		optClickFile(row,ev){
			var vm = this;
			vm.rowDataFile = row;
			var TuiJian = '',
				recommendIcon = '',
				modifyShow = false,
				deleteShow = true;
			var activeName = 'upgrade';
			var showCls = {
    			'upgrade':'CODE_GNB_UPGRADE_FILE hidden',
    		}
			if(row.recommend == '1'){//说明此文件是推荐文件
				TuiJian = '<%=rb.getString("QuXiaoTuiJian")%>';
				recommendIcon = 'el-icon el-icon-operation-cancel-recommend';
			}else{
				TuiJian = '<%=rb.getString("TuiJian")%>';
				recommendIcon = 'el-icon el-icon-operation-recommend';
			}
			if( is_super_user == 'true' ){
				modifyShow = true;
			}else{
				deleteShow = false;
			}
			vm.menus_file = [
				{label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:"view"},
				{label:'<%=rb.getString("XiaZai")%>',cls:"el-icon el-icon-operation-download" + " " +showCls[activeName],code:"download"},
				{label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit" + " " +showCls[activeName],code:"modify"},
				{label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete" + " " +showCls[activeName],code:"del"},
				{label:TuiJian,cls:recommendIcon + " " +showCls[activeName],code:"recommend",show: modifyShow}
			]
			vm.$nextTick(function(){
				document.body.click();
				vm.$refs.menu_file.show(ev);
			})
		},
		clickMenuFile(ev){
			var codes = {
					view:this.viewFile,
					modify:this.editFile,
					download:this.downloadFile,
					del:this.delFile,
					recommend:this.recommendFile
			}
			if(codes[ev.code]){
				codes[ev.code](this.rowDataFile)
			}
		},
		handerClose(){
			this.$refs.menu_task.hide();
			this.$refs.menu_file.hide();
		},
		viewFile(row){
			var vm = this;

			vm.importFileType = 'view';
			vm.importViewFlag = true;
			Object.keys(vm.importFileForm).forEach(function(key){
				if(key == 'fileName'){
					vm.importFileForm[key] = row.file_name ? row.file_name : '';
				}else if(key == 'productValue'){
					vm.importFileForm[key] = row.productValue ? row.productValue : '';
				}else{
					if(row[key] != undefined && row[key] != null){
						vm.importFileForm[key] = row[key];
					}
				}
			});
			vm.importFileShow = true;
		},
		editFile(row){
			var vm = this;

			vm.importFileType = 'edit';
			vm.importViewFlag = false;
			Object.keys(vm.importFileForm).forEach(function(key){
				if(key == 'fileName'){
					vm.importFileForm[key] = row.file_name ? row.file_name : '';
				}else if(key == 'productValue'){
					vm.importFileForm[key] = row.productValue ? row.productValue : '';
				}else{
					if(row[key] != undefined && row[key] != null){
						vm.importFileForm[key] = row[key];
					}
				}
			});
			vm.importFileShow = true;
		},
		downloadFile(){
			var vm = this;
			var params = {
					fileName : vm.rowDataFile.file_name,
					fileType : 1
			}
			axios.post("${ctx}/omc/version/file/fileIsExist.action",stringify(params)).then(function(response){
				var data = response.data;
				if(data["success"]){
					var url = '${ctx}/omc/version/file/downLoadFile.action';
					exportByForm(url,params);
				}else{
					vm.$message.error(data["message"])
				}
			})
		},
		delFile(){
			var vm = this;
			var confirmStr = '<%=rb.getString("QueRenShanChu")%>';
			var url = '${ctx}/cell/version/deleteVersionFile.action';
			var params = {
					fileID :vm.rowDataFile.id
			}
			vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
				customClass:"warningConfirm",
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancelButtonText:'<%=rb.getString("QuXiao")%>',
				type:'warning',
				closeOnClickModal:false
			}).then(() => {
				axios.post(url,stringify(params)).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.$message({
    						message:'<%=rb.getString("ChengGong")%>',
    						type:'success',
    					})
                        vm.$refs.file_table.refresh();
					}else{
						vm.$message.error(data["message"])
					}
				}).catch(() => {})
			})
		},
		recommendFile(){
			var vm = this;
			var params = {
					id : vm.rowDataFile.id
			}
			if(vm.rowDataFile.recommend == '0'){
				params.recommend = '1'
			}else{
				params.recommend = '0'
			}
			axios.post("${ctx}/cell/version/updateRecommendStatus.action",stringify(params)).then(function(response){
				var data = response.data;
				if(data["success"]){
					vm.$refs.file_table.refresh();
				}else{
					vm.$messager.error(data["message"]);
				}
			})
		},
		selectCell(selection){
			this.cellData = selection;
		},
		addRbTask(){
			var vm = this;
			vm.slideTitle = '<%=rb.getString("ShengJiHuiTuiRenWu")%>';
    	    vm.slideUrl = '${ctx}/gnb/upgrade/goAddRollbackTask.action';
    	    vm.slideFooter = true;
    	    vm.slideHeader = true;
    	    vm.slidePosition = 'top';
    	    vm.slideHeight = '100%';
    	    vm.slideWidth = '100%';
            vm.slideSubmitLoading = false;
    	    vm.operType = 'addTask';
    	    vm.$refs.slide.showSlide(function(){
    	    	vm.modal = false
				eventBus.$emit('task-init','','addTask');
    	    });
		},
	    loadSuccessTask(data){
	    	this.waitNum = data.properties.watingCount;
	    	this.progressNum = data.properties.inProgressCount;
	    	this.suspendNum = data.properties.suspendCount;
	    	this.endNum = data.properties.endCount;
	    },
	    loadsuccessResult(data){
	    	this.sucNum = data.properties.successCount;
	    	this.failNum = data.properties.failureCount;
	    },
	    queryResult(){
	    	if(this.list_name == "software"){
	    		Object.assign(this.query_result_params,this.query_result_form)
	    		this.query_result_params.rd = Math.random();
	    	}else{
	    		Object.assign(this.query_result_params_rb,this.query_result_form_rb)
	    		this.query_result_params_rb.rd = Math.random();
	    	}
	    	
	    },
	    clickList(tab){
	    	if(tab.name == "software"){
	    		this.$refs.upgrade_task_table.refresh();
	    		this.$refs.upgrade_result_table.refresh();
	    		this.taskUrlRb = "";
	    		this.resultUrlRb = "";
	    	}else{
	    		this.taskUrlRb = "${ctx}/task/upgrade/getUpgradeTaskList.action";
	    		this.resultUrlRb = "${ctx}/task/upgrade/getUpgradeDeviceList/rollback.action";
	    	}
	    },
	    exportResult(){
	    	var vm = this;
	    	var url = "";
	    	var params = {isGnb:1}
	    	if(vm.list_name == "software"){
	    		url = "${ctx}/task/upgrade/exportUpgradeDeviceListToCSV/upgrade.action";
	    		params.timeZone = timeZone;
	    		params.searchText = vm.query_result_form.searchText;
	    	}else if(vm.list_name == "rollback"){
	    		url = "${ctx}/task/upgrade/exportUpgradeDeviceListToCSV/rollback.action";
	    		params.timeZone = timeZone;
	    		params.searchText = vm.query_result_form_rb.searchText;
	    	}
	    	exportByForm(url,params);
	    },
	    towhoFmt(row,column,cellVal,index){
			if(cellVal =='all'){
				return "GA"
			}else if(cellVal == 'beta'){
				return "Beta"
			}else if(cellVal == 'none'){
				return "Test"
			}
		},
		configFmt(row,column,cellValue,index){
	    	if(cellValue == 'true'){
	    		return '<%=rb.getString("Fou")%>'
	    	}else{
	    		return '<%=rb.getString("Shi")%>'
	    	}
	    },
	    restartTask(type,taskId,snCode){
	    	var vm = this;
	    	var result = [];
	    	if(type == 'single'){
	    		result = [{[taskId]:snCode}]
	    	}else{
	    		vm.selectionDevice.map(function(item){
	    			result.push({[item.taskId]:item.smallCellCode})
	    		})
	    	}
	    	var params = {
	    		taskDeviceList : JSON.stringify(result)
	    	}
			axios.post("${ctx}/task/upgrade/reTryUpgrade.action",stringify(params)).then(function(response){
				var data = response.data;
				if(data) {
					if(data["success"]){
						vm.$message({
							message: '<%=rb.getString("ChengGong")%>',
							type:'success'
						});
						vm.$refs.upgrade_task_table.refresh()
						vm.$refs.upgrade_result_table.refresh()
					}else{
						vm.$message.error(data["message"])
					}
				}
			})
	    },
	    selectDevice(selection){
	    	this.selectionDevice = selection;
	    },
	    choseDevice(row,index){
	    	if(row.result == '3'){return true}
	    	return false;
	    },
		//改变产品类型
		changeProduct(val){
			var vm = this;

			if(vm.query_cell_params.productValue == val)return
			vm.$refs.upgrade_cell_table.clearSelection();
			vm.query_cell_params.productValue = val;
		},
		dateChange(val) {
			var vm = this;
			vm.dateValue = val;
			if(val != null){
				this.query_task_params.startTime = vm.dateValue[0];
    			this.query_task_params.endTime = vm.dateValue[1];
			}
		},
		dateChangeSoft(val) {
			var vm = this;
			vm.dateValueRb = val;
			if(val != null){
				this.query_task_params_rb.startTime = vm.dateValueRb[0];
    			this.query_task_params_rb.endTime = vm.dateValueRb[1];
			}
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
			Object.assign(vm.query_cell_params, params);
		},
		// 清除筛选
		clearFilterClick(){
			var vm = this,
				params = {};
			vm.advancedQueryItemList.map((items)=>{
				if(items.isShow && items.isShow== true ){
					items.selectVal = '';
					params[items.value] = '';
				}
			})
			Object.assign(vm.query_cell_params, params);
			document.body.click();
		},
		// 升级文件选择
		importFileChange(file, fileList) {
			var vm = this,
				fileSize = file.size;
			if(fileSize <= vm.multipartMaxFileSize){
				vm.fileErrorData = '';
				vm.importFileForm.file = file.raw;
				vm.importFileForm.fileName = file.name;
			}else{
				var maxFileSizeMB = Number(vm.multipartMaxFileSize / 1024 / 1024).toFixed(0),
					fileSizeMB = Number(fileSize / 1024 / 1024).toFixed(0),
					messageStr = '<%=rb.getString("CollectLogSizeExceedOne")%>' + fileSizeMB + '<%=rb.getString("CollectLogSizeExceedTwo")%>' + maxFileSizeMB + '<%=rb.getString("CollectLogSizeExceedThree")%>';
				vm.$message({
					type: 'error',
					message: messageStr
				});
				vm.$refs.importFile.clearFiles();
			}
			
			
		},
		/**
		* 文件上传之前
		* @param file{object}   文件信息
		*/ 
		beforeUpload(file){
			var vm = this;
			var fileName = file.name,fileSize = file.size;
			var fd = new FormData(),
				config = {
					headers: { 'Content-Type': 'multipart/form-data' },
					onUploadProgress:(ev)=>{
						if(ev.lengthComputable || ev.event.lengthComputable) {
							var total = ev.total,
								loaded = ev.loaded,
								percent = 100*loaded/total;
							$('#progressUploadFile').progressbar('setValue', percent.toFixed(2));
						}
					}
				};
			fd.append('uploadFile',file); //文件流
			fd.append('newFileName',fileName); //文件流
			fd.append('fileSize',fileSize);//文件大小
			fd.append('desc',vm.importFileForm.desc);//描述
			fd.append('md5','');
			fd.append('token',omctoken);
			fd.append('isGnb',1);
			fd.append('to_who','all');
			
			fd.append('product',vm.importFileForm.productValue);
			
			fd.append('version',vm.importFileForm.version);
			fd.append('recommend',vm.importFileForm.recommend);
			fd.append('fileType','upgrade');
			vm.fileErrorData = vm.$refs.importFile.uploadFiles[0];
			$('#progressUploadFile').progressbar('setValue', 0);// 将进度条进度置为0
			$("#winUploadPro").window("open");// 打开进度条窗口
			axios.post("${ctx}/cell/version/uploadVersionFile.action",fd,config).then(function(response){
				var data = response.data
				$("#winUploadPro").window("close");// 关闭进度条窗口
				if(data["MD5"]){
					vm.fileErrorData = '';
					$.messager.alert('<%=rb.getString("TiShi")%>','<%=rb.getString("ShangChuanChengGong")%><%=rb.getString("DouHao")%><%=rb.getString("WenJianMD5Zhi")%><%=rb.getString("MaoHao")%>'+data["MD5"]);
					vm.$refs.file_table.refresh();
					vm.rightBoxClose();
				}else{
					vm.$message.error(data["message"]);
				}
			})
			
			return false;
		},
		//发送请求，校验device文件内容 
		checkImportFile(res, file) {    
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
			var fileList = vm.$refs.importFile.uploadFiles;
			fileList.forEach(function (file) {
				file.status = 'ready';
			})
		},
		importFileSelect() { 
			var vm = this;
			vm.$refs.importFile.clearFiles();
			vm.$refs['file_up'].click();
		},
		getVersion(val){
			var vm = this;

			var fileFmt = this.difFileObj['upgrade'].fileFmt;

			if(val != '' && vm.fileFormatMatch(val,fileFmt)){
				var pathSplit = val.split(/\\/);
				var filename = pathSplit[pathSplit.length - 1];
				if(filename.substring(filename.length-6) == 'tar.gz'){
					vm.importFileForm.version = filename.substring(0,filename.length-7);
				}else{
					vm.importFileForm.version = filename.substring(0,filename.lastIndexOf("."));
				}
				vm.$refs.importFileForm.validateField('version')
			}
		},
		// 文件校验
		fileFormatMatch(str,regs){
			var regsArr = regs.toLowerCase().split(",");
			if(str.substring(str.length-6) == 'tar.gz'){
				var suffix = "tar.gz";
			}else{
				var suffix = str.substring(str.lastIndexOf(".")+1).toLowerCase();
			}
			if(regsArr.indexOf(suffix)>-1){
				return true;
			}else{
				return false;
			}
		},
		// 升级文件导入确定
		importFileSubmit(){
			var vm = this;
			vm.$refs.importFileForm.validate((valid) => {
				if(valid){
					if(vm.importFileType == 'add'){
						if(vm.fileErrorData){
							vm.$refs.importFile.uploadFiles.push(vm.fileErrorData);
						}
						vm.$refs.importFile.submit();
					}else{
						vm.fileImportEditSubmit()
					}
				}
			})
		},
		// 升级文件 修改提交
		fileImportEditSubmit(){
			var vm = this,
				params={
					versionId:vm.rowDataFile.id,
					fileName:vm.importFileForm.fileName,
					productValue:vm.importFileForm.productValue,
					version:vm.importFileForm.version,
					recommend:vm.importFileForm.recommend,
					desc:vm.importFileForm.desc,
					isGnb: 1,
					fileType: '0'
				};
			axios.post("${ctx}/cell/version/goModifyDeviceVersionFileInfo.action",stringify(params)).then(function(response){
				var data = response.data;
				if(data) {
					if(data["success"]){
						vm.$message({
							message: '<%=rb.getString("ChengGong")%>',
							type:'success'
						});
						vm.$refs.file_table.refresh();
						vm.rightBoxClose();
					}else{
						vm.$message.error(data["message"])
					}
				}
			}).catch(function(error){})
		},
		// 升级文件导入 关闭
		rightBoxClose(){
			var vm = this,
				params = {
					productValue:'',
					file:'',
					fileName:'',
					version:'',
					recommend:'1',
					desc:'',
					to_who:'all',
				};
			vm.importFileShow = false;
			Object.assign(vm.importFileForm,params)

		},
		fileTypeClick(event){
			var vm = this;
			event.preventDefault();
		},
	},
	computed: {
		rightOutBoxTitle() {
			var vm = this,
				codes={
					'add':'File Import',
					'view':'File Information',
					'edit':'File Modify'
				};

			return codes[this.importFileType];
		},
		optBtnShow() {
			return writableMap['CODE_GNB_UPGRADE_IMAGE'] == true;
		},
	},
	watch:{
		list_name:function(val){
			if(val == "software"){
	    		this.$refs.upgrade_task_table.refresh();
	    		this.$refs.upgrade_result_table.refresh();
	    		this.taskUrlRb = "";
	    		this.resultUrlRb = "";
	    	}else{
	    		this.taskUrlRb = "${ctx}/task/upgrade/getUpgradeTaskList.action";
	    		this.resultUrlRb = "${ctx}/task/upgrade/getUpgradeDeviceList/rollback.action";
	    	}
		},
		dateValue(newVal){
			var vm = this;
			if(!newVal){
				newVal = [];
				vm.query_task_params.startTime = '';
    			vm.query_task_params.endTime = '';
			}
		},
		dateValueRb(newVal){
			var vm = this;
			if(!newVal){
				newVal = [];
				vm.query_task_params_rb.startTime = '';
    			vm.query_task_params_rb.endTime = '';
			}
		},
		"importFileForm.fileName":function(val){
			var vm = this;
			if(vm.importFileType == 'add'){
				this.getVersion(val)
			}
		},
	},
	mounted(){
		this.init();
	}
})
</script>