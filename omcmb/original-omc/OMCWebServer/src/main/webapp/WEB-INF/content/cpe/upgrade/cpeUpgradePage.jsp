<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>

	#cpeUpgradePage .el-badge{
	    position:relative;
    }
    #cpeUpgradePage .el-badge__content{
        position:absolute;
        top:6px;
        right:0px;
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
    #cpeUpgradePage .el-icon-star-badge:before{
        color:#F3916C;
    }
	#cpeUpgradePage .container .cmenu{
		z-index: 361!important;
	}
	#upsUpgradeSlide{
		z-index: 2002!important;
	}
	#cpeUpgradePage .linkStyle {
		margin-left:5px;
		color:#4D84FF;
		cursor:pointer;
	}

	#cpeUpgradePage .deviceTable{
		height: calc(50% - 5px);
	}
	#cpeUpgradePage .flex-form {
		display: flex;
		flex-wrap: wrap;
	}
	#cpeUpgradePage .flex-form .el-form-item {
		margin-right: 100px;
		margin-bottom: 10px;
	}
	#cpeUpgradePage .upgradeTaskBox{
		box-sizing: border-box;
		height: calc(50% - 5px);
		background-color: #FFFFFF;
		overflow: auto;
	}
	#cpeUpgradePage .upgradeTaskBox .el-tabs{
		height: 100%;
	}
	#cpeUpgradePage .upgradeTaskBox .el-tabs__header{
		border-top: none;
		border-bottom: 1px solid #E9E9E9 !important;
	}
	#cpeUpgradePage .upgradeTaskBox .el-tabs__item{
		font-size:14px;
	}
	#cpeUpgradePage .upgradeTaskBox .el-tabs--top{
		border:none;
	}
	#cpeUpgradePage .upgradeTaskBox .el-tabs__nav-scroll{
		margin-left:10px;
	}
	
	#cpeUpgradePage .headTipCls{
		position: absolute;
		left: 520px;
		top:17px;
		font-size: 12px;
		z-index: 99;
	}
	#cpeUpgradePage .headTipCls>div{
		display: inline-block;
	}
	#cpeUpgradePage .headTipCls .fontTitle{
		color: #666666;
		padding: 0px 5px;
	}
	#cpeUpgradePage .headTipCls .fontInfo{
		color: #333333;
		font-weight: bold;
	}
	
	#cpeUpgradePage .taskHeadBox{
		position: absolute;
		top:15px;
		right: 0;
		display: flex;
		z-index: 99;
	}
	#cpeUpgradePage .taskHeadBox .statisticsDiv{
		height: 16px;
		line-height: 16px;
		display: flex;
		overflow: hidden;
		font-size: 14px;
	}
	#cpeUpgradePage .statisticsDiv > div:first-child{
		padding: 0px 0 0 10px;
		color: #4D84FF;
		background: #FFFFFF;
		height: 16px;
		line-height: 16px;
	}
	#cpeUpgradePage .statisticsDiv > div:last-child{
		padding: 0px 10px;
	}
	#cpeUpgradePage .statisticsDiv .el-icon, #cpeUpgradePage .statisticsSuccessDiv .el-icon, #cpeUpgradePage .statisticsFailDiv .el-icon{
		margin-right: 6px;
		font-size: 14px;
	}
	#cpeUpgradePage .resultHeadBox{
		position: absolute;
		top:17px;
		right: 60px;
		display: flex;
		z-index: 99;
	}
	#cpeUpgradePage .resultHeadBox .statisticsSuccessDiv{
		height: 14px;
		line-height: 14px;
		display: flex;
		font-size: 14px;
		border-right: 1px solid #DFE2EE;
	}
	#cpeUpgradePage .statisticsSuccessDiv .el-icon::before{
		color:#67D972;
	}
	#cpeUpgradePage .statisticsSuccessDiv > div:first-child{
		padding: 0px 10px;
		color: #67D972;
	}
	#cpeUpgradePage .statisticsSuccessDiv > div:last-child{
		padding: 0px 10px 0 0;
	}
	#cpeUpgradePage .resultHeadBox .statisticsFailDiv{
		height: 14px;
		line-height: 14px;
		display: flex;
		font-size: 14px;
	}
	#cpeUpgradePage .statisticsFailDiv .el-icon::before{
		color: #E88282;
		content:'\e6fb';
	}
	#cpeUpgradePage .statisticsFailDiv > div:first-child{
		padding: 0px 10px;
		color: #E88282;
	}

	#cpeUpgradePage .fileTableBoxCls{
		height: calc(100% - 2px);
		border: 1px solid #D5DCEC; 
		border-radius: 0 0 8px 8px;
		background: #FFFFFF;
	}
	#cpeUpgradePage .device_item{
		position:relative;
	}
	#cpeUpgradePage .device_item .el-ctable-toolbar {
		padding: 0 !important;
	}
	#cpeUpgradePage .list_item .el-tabs__item{
		font-size:14px;
	}
	#cpeUpgradePage .list_item .el-tabs--top{
		border:none;
	}
	#cpeUpgradePage .list_item{
		position:relative;
	}
	#cpeUpgradePage .file_item .queryGroup{
		margin-left:20px;
	}
	#cpeUpgradePage .file_item .el-ctable-toolbar {
		padding: 10px 0 0 !important;
	}
	#cpeUpgradePage {
		border: 0;
		background: #F6F7FB;
	}
	#cpeUpgradePage .list_item .el-tabs__header {
		border: 0;
	}
	
	#cpeUpgradePage .commonTabsTop .el-tabs__header {
		border: 1px solid #D5DCEC;
		border-bottom: 0;
		border-radius: 8px 8px 0 0;
		padding: 0 20px;
	}
	#cpeUpgradePage .list_item .el-tabs__header {
		border: 0;
	}
	#cpeUpgradePage .container .el-ctable-toolbar{
		padding: 0px !important;
	}
	#cpeUpgradePage .fileTableItemCls{
		height: calc(100% - 38px);
	}
	#cpeUpgradePage .el-radio.is-bordered,
	.addDeviceDialog .el-radio.is-bordered{
		height: 30px;
		padding: 7px 20px 0 10px;
	}
	#cpeUpgradePage .el-radio.is-bordered+.el-radio.is-bordered,
	.addDeviceDialog .el-radio.is-bordered+.el-radio.is-bordered{
		margin-left: 15px;
	}
</style>
<!-- cpe升级 -->
<div class="pageDefault" id='cpeUpgradePage'>
	<div class="container">
        <el-tabs class="fit commonTabsTop" v-model="activeName" style='height:calc(100% - 2px)'>
            <el-tab-pane name="upgrade" label='<%=rb.getString("ShengJi")%>' style='border-top: 1px solid #D5DCEC'>
				<div class="deviceTable device_item">
					<el-ctable class='commonTableBorder'
						:url="deviceTableUrl"
						:query-params="queryDeviceParams" 
						ref="cpeUpgradedeviceTable" 
						:height="height" 
						@selection-change='deviceSelect'
						:page-size="pageSize" 
						:page-list="pageList" 
						pagination="true">
							<!-- 列表toolbar -->
						<template slot="toolbar">
							<div class='toolbarHeadBtnBoxCls commonQuery' style='margin-bottom: 0;height:45px;'>
								<div class="newIconBoxCls-bt CODE_CPE_UPGRADE_IMAGE hidden" style="right:20px;top:5px;" @click="addUpgradeTask" tip="<%=rb.getString("ShengJi")%>">
									<span class='el-icon el-icon-circle-upgrade'></span>
								</div>
								<el-query type="normal"  @query="queryDevice" placeholder="<%=rb.getString("CPEBianMa")%> / <%=rb.getString("CPEName")%> / <%=rb.getString("HostName")%> / <%=rb.getString("IMSI")%> / PCI"></el-query>
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
						</template>
							<!-- 列表columns -->
						<el-table-column v-if="hasUpgradeRole" type="selection" width="45"></el-table-column>
						<el-table-column prop="connection_status" width="50">
							<template slot-scope="scope">
								<div :class="{
									'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
									'':scope.row.have_connected==2,
									'conn_exc':scope.row.connection_status=='Exception',
									'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
							</template>
						</el-table-column>
						<el-table-column prop="small_cell_code" v-if="false"></el-table-column>
						<el-table-column prop="serial_number" show-overflow-tooltip label="<%=rb.getString("CPEBianMa")%>" min-width="150"></el-table-column>
						<el-table-column prop="imsi" show-overflow-tooltip label="<%=rb.getString("IMSI")%>" min-width="160" ></el-table-column>
						<el-table-column prop="cpe_device_name" show-overflow-tooltip label="<%=rb.getString("CPEName")%>" min-width="220" ></el-table-column>
						<el-table-column prop='ipaddress' label='IP' width="180"></el-table-column>
						<el-table-column prop="mac_address" label="<%=rb.getString("CPEMacAddress")%>" min-width="160" ></el-table-column>
						<el-table-column prop="software_version" show-overflow-tooltip label="<%=rb.getString("BanBen")%>"  min-width="160"></el-table-column>
						<el-table-column prop="model_name" sortable show-overflow-tooltip label="<%=rb.getString("ChanPinXingHao")%>" min-width="160" ></el-table-column>
						<el-table-column prop="cell_name" label="<%=rb.getString("HostName")%>" min-width="140" sortable></el-table-column>
						<el-table-column prop="cell_identity" label="ECI" min-width="80" ></el-table-column>
						<el-table-column prop="pci" label="PCI" min-width="80" ></el-table-column>
						<el-table-column prop="group_name" show-overflow-tooltip label="<%=rb.getString("SheBeiZu")%>" min-width="160" ></el-table-column>
						<el-table-column prop="moduleName" show-overflow-tooltip label="<%=rb.getString("MoKuaiMingCheng")%>" min-width="110" ></el-table-column>
						<el-table-column prop="moduleVersion" show-overflow-tooltip label="<%=rb.getString("MoKuaiBanNen")%>" min-width="140" ></el-table-column>
					</el-ctable>
				</div>
				
				<div class="upgradeTaskBox list_item" style='border: 1px solid #D5DCEC; border-radius: 8px; margin-top:10px;'>
					<el-tabs v-model="upgrade_activeName" class='newTabs'>
						<el-tab-pane label="<%=rb.getString("RenWuLieBiao")%>" name="task">
							<div class="taskHeadBox">
								<div class="statisticsDiv">
									<div><span class="el-icon el-icon-status-waiting1"></span><%=rb.getString("DengDai")%></div>
									<div>{{statisticTaskStatus.WaitingNum}}</div>
								</div>
								<div class="statisticsDiv">
									<div><span class="el-icon el-icon-status-inProgress"></span><%=rb.getString("JinXingZhong")%></div>
									<div>{{statisticTaskStatus.ProcessingNum}}</div>
								</div>
								<div class="statisticsDiv">
									<div><span class="el-icon el-icon-status-suspend"></span><%=rb.getString("ZanTing")%></div>
									<div>{{statisticTaskStatus.SuspendedNum}}</div>
								</div>
								<div class="statisticsDiv">
									<div><span class="el-icon el-icon-status-terminate"></span><%=rb.getString("JieShu")%></div>
									<div>{{statisticTaskStatus.EndNum}}</div>
								</div>
							</div>
							<el-ctable id="cpeUpgradeTaskTable" time=6  @load-success="taskTableLoadSuccess" ref="cpeUpgradeTaskListTable" :url="taskListTableUrl" :query-params="queryTaskParams" :page-size="20" :pagination=true height="100%">
								<template slot="toolbar">
									<div class='toolbarHeadBtnBoxCls commonQuery' style="height:45px;">
										<el-query type="normal" @query="queryTask" placeholder="<%=rb.getString("RenWuMingCheng")%>"></el-query>
										<el-date-picker style='margin-left: 20px;' 
											v-model="timeRange"
											type="datetimerange"
											value-format="yyyy-MM-dd HH:mm:ss"
											range-separator="——"  
											@change="dateChange"
											start-placeholder='<%=rb.getString("KaiShiShiJian")%>' 
											end-placeholder='<%=rb.getString("JieShuShiJian")%>'>
										</el-date-picker>
									</div>
								</template>
								<el-table-column label="" width="30" class-name="no-text-tips">
									<template slot-scope="scope"><!-- 将元素或组件表示为作用域插槽          -->
										<div class="el-icon el-icon-operation-more" @click="taskOptClick(scope.row,event)" v-clickoutside="hideMenus"></div>
									</template>
								</el-table-column>
								<el-table-column prop="TASK_ID"  v-if="false"></el-table-column>
								<el-table-column prop="TYPE" v-if="false"></el-table-column>
								<el-table-column prop="TASK_NAME" show-overflow-tooltip label="<%=rb.getString("RenWuMingCheng")%>"  min-width="200"></el-table-column>
								<el-table-column prop="CREATE_USER" label="<%=rb.getString("CaoZuoRen")%>"  min-width="100" ></el-table-column>
								<el-table-column prop="CREATE_TIME" label="<%=rb.getString("CaoZuoShiJian")%>" min-width="180"></el-table-column>
								<el-table-column prop="FILE_NAME" show-overflow-tooltip label="<%=rb.getString("WenJianMing")%>" min-width="200"></el-table-column>
								<el-table-column prop="VERSION" label="<%=rb.getString("BanBen")%>" min-width="200"></el-table-column>
								<el-table-column prop="upgradeMode" label="Upgrade Type" min-width="160" >
									<template slot-scope="scope">
										<div v-if="scope.row.upgradeMode == 'image'">Image Upgrade File</div>
										<div v-if="scope.row.upgradeMode == 'module'">Module Upgrade File</div>
									</template>
								</el-table-column>
								<el-table-column prop="moduleName" show-overflow-tooltip label="<%=rb.getString("MoKuaiMingCheng")%>" min-width="110" ></el-table-column>
								<el-table-column prop="TASK_STATUS" label="<%=rb.getString("ZhuangTai")%>" min-width="120">
									<template slot-scope="scope">
										<div v-html="taskTableStatus(scope.row.TASK_STATUS)"></div>
									</template>
								</el-table-column>
								<el-table-column prop="TASK_PROGRESS" label="<%=rb.getString("JinDu")%>" min-width="100"></el-table-column>
								<el-table-column prop="TASK_RESULT" label="<%=rb.getString("JieGuo")%>" min-width="100" :formatter="taskTableResult"></el-table-column>
								<el-table-column prop="START_TIME" label="<%=rb.getString("KaiShiShiJian")%>" min-width="180"></el-table-column>
								<el-table-column prop="END_TIME" label="<%=rb.getString("JieShuShiJian")%>" min-width="180"></el-table-column>
							</el-ctable>
							<el-cmenu ref="taskMenu" :data="taskMenus" @click="taskMenuClick"></el-cmenu>
						</el-tab-pane>
						<el-tab-pane label="<%=rb.getString("SheBeiLieBiao")%>" name="device"> 
							<div class="resultHeadBox">
								<div class="statisticsSuccessDiv">
									<div><span class="el-icon el-icon-circle-success"></span><%=rb.getString("ChengGong")%></div>
									<div>{{statisticDeviceResult.SucNum}}</div>
								</div>
								<div class="statisticsFailDiv">
									<div><span class="el-icon el-icon-circle-close"></span><%=rb.getString("ShiBai")%></div>
									<div>{{statisticDeviceResult.FailNum}}</div>
								</div>
							</div>
							<!--:url="queryResultUrl"  :data-->
							<el-ctable 
								id="cpeUpgradeResultTable"
								ref="queryResultTable" 
								:url="queryResultUrl" 
								:query-params="queryResultParams"
								@load-success="deviceTableLoadSuccess"
								height="100%"
								time=6
								:page-size="20" 
								:pagination=true>
								<template slot="toolbar">
									<div class='toolbarHeadBtnBoxCls commonQuery' style='height:45px; position: relative;'>
										<div class="newIconBoxCls-bt" style="right:10px;top:10px;" @click="exportResultTable" tip="<%=rb.getString("DaoChu")%>">
											<span class='el-icon el-icon-operation-export'></span>
										</div>
										<el-query type="normal" @query="queryCpeUpgradeResult" placeholder="<%=rb.getString("CPEBianMa")%> / <%=rb.getString("CPEName")%> / <%=rb.getString("RenWuMingCheng")%> / <%=rb.getString("IMSI")%>"></el-query>
									</div>
								</template>
								<el-table-column prop="ID" v-if="false"></el-table-column>
								<el-table-column v-if="hasUpgradeRole" label='' width="30" class-name="no-text-tips">
									<template slot-scope = "scope">
										<div v-if="scope.row.PROGRESS_RESULT == '3'" class="el-icon el-icon-operation-restart" @click="restartTask('single',scope.row.taskId,scope.row.cpeCode)" style="cursor: pointer;"></div>
									</template>
								</el-table-column>
								<el-table-column prop="SERIAL_NUMBER" show-overflow-tooltip label="<%=rb.getString("CPEBianMa")%>" min-width="160"></el-table-column>
								<el-table-column prop="imsi" show-overflow-tooltip label="<%=rb.getString("IMSI")%>" min-width="160" ></el-table-column>
								<el-table-column prop="CPE_NAME" show-overflow-tooltip label="<%=rb.getString("CPEName")%>"  min-width="100"></el-table-column>
								<el-table-column prop="TASK_NAME" show-overflow-tooltip label="<%=rb.getString("RenWuMingCheng")%>"  min-width="280"></el-table-column>
								<el-table-column prop="ORI_VERSION" show-overflow-tooltip label="<%=rb.getString("ChuShiBanBen")%>" min-width="200"></el-table-column>
								<el-table-column prop="TARGET_VERSION" show-overflow-tooltip label="<%=rb.getString("ShengJiBanBen")%>" min-width="200"></el-table-column>
								<el-table-column prop="PROGRESS_STATUS" label="<%=rb.getString("ZhuangTai")%>" min-width="120">
									<template slot-scope="scope">
										<div v-html="resultTableStatus(scope.row.PROGRESS_STATUS)"></div>
									</template>
								</el-table-column>
								<el-table-column prop="PROGRESS_RESULT" label="<%=rb.getString("JieGuo")%>" min-width="110" :formatter="resultTableResult"></el-table-column>
								<el-table-column prop="FAILURE_REASON" show-overflow-tooltip label="<%=rb.getString("PCILOCKShiBaiYuanYin")%>" min-width="120"></el-table-column>
								<el-table-column prop="START_TIME" label='<%=rb.getString("KaiShiShiJian")%>' min-width="160" ></el-table-column>
								<el-table-column prop="END_TIME" label='<%=rb.getString("JieShuShiJian")%>'  min-width="160"></el-table-column>
							</el-ctable>
						</el-tab-pane>
					</el-tabs>
				</div>
            </el-tab-pane>
			<el-tab-pane name="file" label="<%=rb.getString("WenJian")%>" class='file_item'>
				<div class="fileTableBoxCls">
					<el-radio-group size="mini" v-model='fileTypeNum' class="commonRadioButton" @change="fileTypeClick" style='margin:10px 10px 0px;display:inline-block;'>
						<el-radio-button label="image"><%=rb.getString("IMAGE")%></el-radio-button>
						<el-radio-button label="mid"><%=rb.getString("ZhongJianBanBen")%></el-radio-button>
						<el-radio-button label="module"><%=rb.getString("MoKuaiBanBen")%></el-radio-button>
					</el-radio-group>
					<!-- 系统升级文件 表格组件 :url="systemFileTableUrl" -->
					<div v-if="fileTypeNum == 'image'" class="fileTableItemCls">
						<el-ctable 
							:url="systemFileTableUrl"
							:query-params="system_params" 
							ref="cpeSystemUpgradeFileTable" 
							id="cpeSystemUpgradeFileTable"
							:height="height" 
							:page-size="pageSize" 
							:page-list="pageList" 
							pagination="true">
								<!-- 列表toolbar -->
							<template slot="toolbar">
								<div class='toolbarHeadBtnBoxCls commonQuery' style='margin-bottom: 0;height:45px;'>
									<div class="newIconBoxCls-bt CODE_CPE_UPGRADE_FILE hidden" style="right:20px;top:5px;" @click="importUpgradeFile" tip="<%=rb.getString("DaoRuWenJian")%>">
										<span class='el-icon el-icon-operation-import'></span>
									</div>
									<el-query type="normal" @query="querySystemUpgradeFile" placeholder="<%=rb.getString("BanBen")%>"></el-query>
								</div>
							</template>
								<!-- 列表columns -->
							<el-table-column prop="op" label=" " width="40" align="center">
								<template slot-scope="scope">
									<div class="el-icon el-icon-operation-more" v-clickoutside="hideMenus" @click="optClick(scope.row,event)"></div>
								</template>
							</el-table-column>
							<el-table-column prop="file_name" label="<%=rb.getString("WenJianMing")%>" min-width="150"></el-table-column>
							<el-table-column prop="version" label="<%=rb.getString("BanBen")%>" min-width="250">
								<template slot-scope="scope">
									<div class='el-badge'><span>{{scope.row.version}}</span><span v-show="scope.row.recommend == '1'" class='el-badge__content el-icon el-icon-star-badge'></span></div>
								</template>
							</el-table-column>
							<el-table-column prop="model_name" label="<%=rb.getString("ChanPinXingHao")%>" min-width="150" show-overflow-tooltip></el-table-column>
							<el-table-column prop="size" label="<%=rb.getString("WenJianDaXiao")%>" min-width="120"></el-table-column>
							<el-table-column v-if="isCloudCore=='true'?true:false" label='<%=rb.getString("BanBenLeiXing")%>' min-width="200" prop="toWho" key="toWho">
								<template slot-scope="scope">
									<div v-html="towhoFormat(scope.row.toWho)"></div>
								</template>
							</el-table-column>
							<el-table-column prop="upload_time" label="<%=rb.getString("ShangChuanShiJian")%>" min-width="140"></el-table-column>
						
						</el-ctable>
					</div>
					<!-- 中间版本升级文件 表格组件  -->
					<div v-if="fileTypeNum == 'mid'">
						<el-ctable
							:url="middleFileTableUrl"
							:query-params="middle_params" 
							ref="middleFileTable" 
							id="middleFileTable"
							:height="height" 
							:page-size="pageSize" 
							:page-list="pageList" 
							pagination="true">
								<!-- 列表toolbar -->
							<template slot="toolbar">
								<div class='toolbarHeadBtnBoxCls commonQuery' style='margin-bottom: 0;height:45px;'>
									<div class="newIconBoxCls-bt CODE_CPE_UPGRADE_FILE hidden" style="right:20px;top:5px;" @click="importUpgradeFile" tip="<%=rb.getString("DaoRuWenJian")%>">
										<span class='el-icon el-icon-operation-import'></span>
									</div>
									<el-query type="normal" @query="queryMiddleUpgradeFile" placeholder="<%=rb.getString("BanBen")%>"></el-query>
								</div>
							</template>
								<!-- 列表columns -->
							<el-table-column prop="op" label=" " width="40" align="center">
								<template slot-scope="scope">
									<div class="el-icon el-icon-operation-more" v-clickoutside="hideMenus" @click="optClick(scope.row,event)"></div>
								</template>
							</el-table-column>
							<el-table-column prop="version" label="<%=rb.getString("BanBen")%>" min-width="250"></el-table-column>
							<el-table-column prop="modelName" label="<%=rb.getString("ChanPinXingHao")%>" min-width="150"></el-table-column>
							<el-table-column prop="oriVersion" label="<%=rb.getString("ChuShiBanBen")%>" width="150" show-overflow-tooltip></el-table-column>
							<el-table-column prop="fileSize" label="<%=rb.getString("WenJianDaXiao")%>" min-width="120"></el-table-column>
							<el-table-column prop="uploadTime" label="<%=rb.getString("ShangChuanShiJian")%>" min-width="140"></el-table-column>
						
						</el-ctable>
					</div>
					<!-- 模块版本升级文件 表格组件 -->
					<div v-if="fileTypeNum == 'module'">
						<el-ctable 
							:url="moduleFileTableUrl" 
							:query-params="module_params" 
							ref="moduleFileTable" 
							id="moduleFileTable"
							:height="height" 
							:page-size="pageSize" 
							:page-list="pageList" 
							pagination="true">
								<!-- 列表toolbar -->
							<template slot="toolbar">
								<div class='toolbarHeadBtnBoxCls commonQuery' style='margin-bottom: 0;height:45px;'>
									<div class="newIconBoxCls-bt CODE_CPE_UPGRADE_FILE hidden" style="right:20px;top:5px;" @click="importUpgradeFile" tip="<%=rb.getString("DaoRuWenJian")%>">
										<span class='el-icon el-icon-operation-import'></span>
									</div>
									<el-query type="normal" @query="queryModuleUpgradeFile" placeholder="<%=rb.getString("BanBen")%>"></el-query>
								</div>
							</template>
								<!-- 列表columns -->
							<el-table-column prop="op" label=" " width="40" align="center">
								<template slot-scope="scope">
									<div class="el-icon el-icon-operation-more" v-clickoutside="hideMenus" @click="optClick(scope.row,event)"></div>
								</template>
							</el-table-column>
							<el-table-column prop="version" label="<%=rb.getString("BanBen")%>" min-width="250"></el-table-column>
							<el-table-column prop="moduleName" label="<%=rb.getString("MoKuaiMingCheng")%>" width="150" show-overflow-tooltip></el-table-column>
							<el-table-column prop="destVersion" label="<%=rb.getString("MuBiaoBanBen")%>" width="150" show-overflow-tooltip></el-table-column>
							<el-table-column prop="fileSize" label="<%=rb.getString("WenJianDaXiao")%>" min-width="120"></el-table-column>
							<el-table-column prop="uploadTime" label="<%=rb.getString("ShangChuanShiJian")%>" min-width="140"></el-table-column>
						</el-ctable>
					</div>
				</div>
                
            </el-tab-pane>
            <!-- 菜单 -->
            <el-cmenu ref="menu" @click="menuClick" :data="menus"></el-cmenu>
        </el-tabs>
		<!-- slide -->
		<el-slide ref="upgradeSlide" id="upgradeSlide" :url='slideUrl' :title="slideTitle" :footer="slideFooter" :header="slideHeader" :position="slidePosition" class='commonBorderSlide'
			:height="slideHeight"  :width='slideWidth' :subloading="slideSubmitLoading" @ok="submitSlide"  @cancel="closeSlide" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
		</el-slide>
		
    </div>
</div>
<script type="text/javascript">
var cpeUpgrade = new Vue({
	el:'#cpeUpgradePage',
	data(){
		return {
			activeName:'upgrade',
            system_params:{
                timeZone:timeZone,
                searchText:'',
            },
			middle_params:{
                timeZone:timeZone,
                version:'',
            },
			module_params:{
                timeZone:timeZone,
                version:'',
            },
            systemFileTableUrl:'${ctx}/cell/version/queryfileInfosList.action?file_type=9',
			middleFileTableUrl:'${ctx}/cell/version/queryMidFileInfos.action',
			moduleFileTableUrl:'${ctx}/cell/version/queryModuleFileInfos.action',

			deviceTableUrl:"${ctx}/cell/cpeinfos/queryCpeInfosListForCpe.action",
			productValue: "1", //产品类型
			productSelectValue: '',
			selection:[],
			
			queryDeviceParams:{
				searchText:'',
				serial_number :'', 
				imsi:'',
				host_name :'', 
				group_id :'', 
				software_version:'',
				model_name:'',
				cell_name: '',
				pci:''
			},
			queryDeviceForm:{
				serial_number :'', //高级查询基站编码
				imsi:'',
				host_name :'', //高级查询基站名称
				group_id :'', //高级查询选择的设备组
				software_version:'',//高级查询选中的版本 
				model_name:'',//高级查询选中的model 
				cell_name: '',
				pci:''
			},
			taskListTableUrl:'${ctx}/task/upgrade/cpe/getUpgradeTaskListForCpe.action',
			upgrade_activeName:'task',
			queryTaskParams:{
				searchText:'',
				timeZone:timeZone,
				taskName:'',
				startTime:'',
				endTime:'',
			},
			timeRange:[],
			queryTaskForm:{
				taskName:'',
				startTime:'',
				endTime:'',
			},
			queryResultUrl:'${ctx}/task/upgrade/cpe/getUpgradeTaskProgressForCpe.action',
			queryResultParams:{
				timeZone: timeZone,
				searchText:'',
			},
			fileTypeNum:'image',
			slideUrl:'',
			slideTitle:'',
			slideHeader:'',
			slideFooter:'',
			slidePosition:'',
			slideHeight:'',
			slideWidth:'',
            slideSubmitLoading:'',

            height:'100%',
            pageSize:100,
			pageList:[50,100,200,500],
            menus:[],
			taskMenus:[],
            rowData:'',
			taskRowData:'',
			statisticTaskStatus:{
				WaitingNum: 0,
				ProcessingNum: 0,
				SuspendedNum: 0,
				EndNum: 0
			},
			statisticDeviceResult:{
				SucNum: 0,
				FailNum: 0
			},
			slideOpenType:'',
			advancedQueryItemList:[
				{
					type:'select',
					isShow:true,
					popoverShow:false,
					selectVal:'',
					label:'<%=rb.getString("ChanPinXingHao") %>',
					options:[],
					value:'model_name',
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
		}
	},
    computed:{
		isSuperAdmin() {
			return is_super_user == 'true';
		},
		hasUpgradeRole() {
			return writableMap.CODE_CPE_UPGRADE_IMAGE == true;
		}
    },
	watch:{
		timeRange(newVal){
			var vm = this;
			if(!newVal){
				newVal = [];
				vm.queryTaskParams.startTime = '';
    			vm.queryTaskParams.endTime = '';
			}
		}
	},
	methods:{
		init(){
			var vm = this;
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
		},
		// 任务表格 加载成功回调
		taskTableLoadSuccess(data){
			var vm = this;

			Object.assign(vm.statisticTaskStatus, data.properties);
			
		},
		// 设备表格 加载成功回调
		deviceTableLoadSuccess(data){
			var vm = this;

			Object.assign(vm.statisticDeviceResult, data.properties);
		},
		// 新建设备升级任务
		addUpgradeTask(){
			var vm = this;
			vm.slideOpenType = 'upgradeTask';
			vm.slideHeader = true;
			vm.slideUrl = '${ctx}/task/upgrade/cpe/goAddTaskForCpe.action';
			vm.slideFooter = true;
			vm.slidePosition = 'top';
			vm.slideHeight = '100%';
			vm.slideWidth = '100%';
            vm.slideSubmitLoading = false;
			vm.slideTitle = '<%=rb.getString("XinJianShengJiRenWu")%>';
			vm.$refs.upgradeSlide.showSlide(()=>{
				eventBus.$emit('cpe-upgrade-taskInit','','addTask',vm.productValue,vm.selection);
			})
		},
		// 设备表格选择事件
		deviceSelect(selection){
			var vm = this;

			vm.selection = selection
		},
		// cpe 升级任务结果 导出
		exportResultTable(){
			var vm = this;

			exportByForm("${ctx}/task/upgrade/cpe/exportUpgradeProgResultForCpe.action",{
				timeZone: timeZone,
				searchText: vm.queryResultParams.searchText
			});
		},
		// cpe 升级任务结果 表格模糊查询
		queryCpeUpgradeResult(val){
			var vm = this;
			vm.queryResultParams.searchText= val;
		},
		// 产品型号点击事件
		productTypeClick(val){
			var vm = this;

			if(vm.productValue == val){
				return
			}else{
				vm.productValue = val;
				vm.getAdvanceCntent();
				vm.deviceTableUrl = "${ctx}/cell/cpeinfos/queryCpeInfosListForCpe.action?forSelect=" + val;
			}
		},
		/**
		 *  任务结果转换
		 * @param cellValue:传入的 结果 数据 进行转换
		*/
		taskTableResult(row,column,cellValue,index){
			var resultObj = {
				"1" : "<%=rb.getString("ChengGong")%>",
				"2" : "<%=rb.getString("BuFenChengGong")%>",
				"3" : "<%=rb.getString("ShiBai")%>",
				"" : "",
			}
			return resultObj[cellValue];
		},
		// 设备结果格式化
		resultTableResult(row,column,cellValue,index){
			var resultObj = {
					"1" : '<%=rb.getString("ChengGong")%>',
					"2" : '<%=rb.getString("ZhongZhi")%>',
					"3" : '<%=rb.getString("ShiBai")%>',
					"" : ''
			}
			return resultObj[cellValue];
		},
		/**
		 *  产品类型标志 转化
		 * @param value:表格数据
		*/
		productFmt(row,column,cellValue,index){
			var code = {
	    			'CPE_VERSION' : 'ODU',
	    			'ODU' : 'ODU',
	    			'CPE_IDU_VERSION' : 'IDU',
	    			'IDU' : 'IDU'
	    	}
	    	return code[cellValue]
		},
		//获取高级查询下拉列表内容
		getAdvanceCntent(){
			var vm = this;
			axios.post('${ctx}/cell/CPE/getCpeSelectFilter.action',stringify({
				forSelect:vm.productValue,
				selectType:"deviceGroup"
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
			}).catch(function(error){});
			axios.post('${ctx}/cell/CPE/getCpeSelectFilter.action',stringify({
				forSelect:vm.productValue,
				selectType:"version"
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
			axios.post('${ctx}/cell/CPE/queryModelNames.action',stringify({
				forSelect:vm.productValue
			})).then(function(response){
				var data = response.data,
					modelNameList=[];

				data.map((item)=>{
					modelNameList.push({label:item.model_name,value:item.model_name})
				})
				vm.advancedQueryItemList.map((items)=>{
					if('model_name' == items.value){
						items.options = modelNameList
					}
				})
				if(modelNameList.length > 0 ){
					vm.queryDeviceParams.model_name = modelNameList[0].value;
				}
				
			}).catch(function(error){})
		
		},
		// 设备表格 模糊查询
		queryDevice(val){
			var vm = this;

			vm.resetDeviceQuery();
			vm.queryDeviceParams.searchText= val;
		},
		// 设备表格 高级查询
		deviceAdvanceQuery(){
			var vm = this;
			Object.assign(vm.queryDeviceParams, vm.queryDeviceForm);
		},
		// 设备表格 高级查询重置
		resetDeviceQuery(){
			var vm = this,
				params = {
					searchText: '',
					serial_number :"", //高级查询基站编码
					imsi:'',
					host_name :"", //高级查询基站名称
					group_id :"", //高级查询选择的设备组
					software_version:"",//高级查询选中的版本 
					model_name:"",//高级查询选中的model 
					cell_name: ""
				};
			
			Object.assign(vm.queryDeviceForm, params);
			Object.assign(vm.queryDeviceParams, params);
		},
		// 升级任务表格 模糊查询
		queryTask(val){
			var vm = this;

			vm.resetTaskQuery();
			vm.queryTaskParams.searchText= val;
		},
		// 升级任务表格 高级查询
		taskAdvanceQuery(){
			var vm = this;
			Object.assign(vm.queryTaskParams, vm.queryTaskForm);
			if(vm.timeRange != null){
				vm.queryTaskParams.startTime = vm.timeRange[0];
				vm.queryTaskParams.endTime = vm.timeRange[1];
			}else{
				vm.queryTaskParams.startTime = '';
				vm.queryTaskParams.startTime = '';
			}
		},
		// 升级任务表格 高级查询重置
		resetTaskQuery(){
			var vm = this,
				params = {
					searchText: '',
					taskName:'',
					startTime:'',
					endTime:'',
				};
			vm.timeRange = [];
			Object.assign(vm.queryTaskForm, params);
			Object.assign(vm.queryTaskParams, params);
		},
		// 文件类型切换
		fileTypeClick(typeVal){
			var vm = this;
			vm.fileTypeNum = typeVal;
		},
		// 升级任务表格 打开操作菜单
		taskOptClick(row,evt){
			var vm = this;

			vm.taskRowData = row;
			var status = row.TASK_STATUS;
			vm.taskMenus = [
                // {label:'<%=rb.getString("KaiShi")%>',cls:"el-icon el-icon-operation-start CODE_CPE_UPGRADE_IMAGE hidden",code:'start'},
                // {label:'<%=rb.getString("ZanTing")%>',cls:"el-icon el-icon-operation-awaiting CODE_CPE_UPGRADE_IMAGE hidden",code:'wait'},
				{label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'info'},
                {label:'<%=rb.getString("ZhongZhi")%>',cls:"el-icon el-icon-operation-terminate CODE_CPE_UPGRADE_IMAGE hidden",code:'end'},
				{label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit CODE_CPE_UPGRADE_IMAGE hidden",code:'mod'},
                {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete CODE_CPE_UPGRADE_IMAGE hidden",code:'del'},
            ];
			initTaskStatus(status,vm.taskMenus);
            vm.$nextTick(function(){
                document.body.click();
                vm.$refs.taskMenu.show(evt)
            })
		},
		/**
		* 菜单点击事件
		* @param ev{object}   行数据
		*/ 
		taskMenuClick(evt){
			var vm = this;
			var codes = {
				start:this.cpeUpgradeTaskStart,	// 开始
				wait:this.cpeUpgradeTaskStop,	// 暂停
				end:this.cpeUpgradeTaskTerminate,	// 终止
	    		info:this.cpeUpgradeTaskInfo,  // 详情
	    		mod:this.cpeUpgradeTaskModify,	// 修改
	    		del:this.cpeUpgradeTaskDel,	// 删除
	    	};
			if(codes[evt.code]){
				codes[evt.code](vm.taskRowData["TASK_ID"],vm.taskRowData["TYPE"],vm.taskRowData["TASK_STATUS"])
			}
		},
		/**
		 * 激活任务
		 * @param id:数据id
		 * @param fileType: 类型
		*/
		cpeUpgradeTaskStart(id,type){
			var vm = this;
			axios.post('${ctx}/task/upgrade/cpe/activeTask.action',stringify({
	    		taskId : id,
	    		type:type
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
					vm.$refs.cpeUpgradeTaskListTable.refresh()
	    		}else{
	    			vm.$message.error(data["message"]) //错误提示信息
	    		}
	    	}) 
		},
		/**
		 * 暂停任务
		 * @param id:数据id
		 * @param fileType: 类型
		*/
		cpeUpgradeTaskStop(id,type){
			var vm = this;
			axios.post('${ctx}/task/upgrade/cpe/suspendTask.action',stringify({
	    		taskId : id,
	    		type:type
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
					vm.$refs.cpeUpgradeTaskListTable.refresh()
	    		}else{
	    			vm.$message.error(data["message"]) //错误提示信息
	    		}
	    	}) 
		},
		/**
		 * 终止任务
		 * @param id:数据id
		 * @param fileType: 类型
		*/
		cpeUpgradeTaskTerminate(id,type){
			var vm = this;
			axios.post('${ctx}/task/upgrade/cpe/terminateUpgradeTask.action',stringify({
	    		taskId : id,
	    		type:type
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
					vm.$refs.cpeUpgradeTaskListTable.refresh()
	    		}else{
	    			vm.$message.error(data["message"]) //错误提示信息
	    		}
	    	}) 
		},
		/**
		 * 查看任务
		 * @param id:当前数据id
		*/
		cpeUpgradeTaskInfo(id){
			var vm = this;
			vm.slideHeader = true;
			vm.slideOpenType = 'upgradeTask';
			vm.slideUrl = '${ctx}/task/upgrade/cpe/goAddTaskForCpe.action';
			vm.slideFooter = false;
			vm.slidePosition = 'top';
			vm.slideHeight = '100%';
			vm.slideWidth = '100%';
            vm.slideSubmitLoading = false;
			vm.slideTitle = '<%=rb.getString("XinXi")%>';
			vm.$refs.upgradeSlide.showSlide(()=>{
				eventBus.$emit('cpe-upgrade-taskInit',id,'taskView',vm.productValue);
			})
		},
		/**
		 * 修改任务
		 * @param id:当前数据id
		*/
		cpeUpgradeTaskModify(id){
			var vm = this;
			vm.slideOpenType = 'upgradeTask';
			vm.slideHeader = true;
			vm.slideUrl = '${ctx}/task/upgrade/cpe/goAddTaskForCpe.action';
			vm.slideFooter = true;
			vm.slidePosition = 'top';
			vm.slideHeight = '100%';
			vm.slideWidth = '100%';
            vm.slideSubmitLoading = false;
			vm.slideTitle = '<%=rb.getString("XiuGai")%>';
			vm.$refs.upgradeSlide.showSlide(()=>{
				eventBus.$emit('cpe-upgrade-taskInit',id,'taskEdit',vm.productValue);
			})
		},
		/**
		 * 删除升级任务
		 * @param id:当前数据id
		*/
		cpeUpgradeTaskDel(id,type){
			var vm = this,
				params = {
					taskId: id,
					type:type
				};
			vm.$confirm('<%=rb.getString("QueRenShanChuRenWu")%>','<%=rb.getString("QueRen")%>',{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		closeOnClickModal:false
	    	}).then(function(){
				axios.post('${ctx}/task/upgrade/cpe/delUpgradeTask.action',stringify(params)).then(function(response){
					var data = response.data;
					if(data) {
						if(data["success"]){
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success'
							});
							vm.$refs.cpeUpgradeTaskListTable.refresh()
						}else{
							vm.$message.error(data["message"])
						}
					}
				}).catch(function(error){})
			});
		},
		restartTask(type,taskId,snCode){
			var vm = this;
	    	var result = [];
	    	if(type == 'single'){
	    		result = [{[taskId]:snCode}]
	    	}
	    	var params = {
	    			taskDeviceList : JSON.stringify(result)
	    	}
			axios.post("${ctx}/task/upgrade/cpe/reTryUpgrade.action",stringify(params)).then(function(response){
				var data = response.data;
				if(data) {
					if(data["success"]){
						vm.$message({
							message: '<%=rb.getString("ChengGong")%>',
							type:'success'
						});
						vm.$refs.cpeUpgradeTaskListTable.refresh();
						vm.$refs.queryResultTable.refresh()
					}else{
						vm.$message.error(data["message"])
					}
				}
			})
		},
		// 菜单关闭
        hideMenus() {
            this.$refs.menu.hide();
			this.$refs.taskMenu.hide();
        },
        // 打开操作菜单
        optClick(row,evt) {
            var vm = this,recommendFlag="";

            vm.rowData = row;
			if(row.recommend == '1'){//说明此文件是推荐文件
				recommendFlag = false;
			}else{
				recommendFlag = true;
			}
			if(vm.fileTypeNum != 'image'){
				vm.menus = [
					{label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'view'},
					{label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit CODE_CPE_UPGRADE_FILE hidden",code:'modify'},
					{label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete CODE_CPE_UPGRADE_FILE hidden",code:'del'},
				];
			}else{
				vm.menus = [
					{label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'view'},
					{label:'<%=rb.getString("XiaZai")%>',cls:"el-icon el-icon-operation-download CODE_CPE_UPGRADE_FILE hidden",code:'download'},
					{label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit CODE_CPE_UPGRADE_FILE hidden",code:'modify'},
					{label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete CODE_CPE_UPGRADE_FILE hidden",code:'del'},
					{label:'<%=rb.getString("TuiJian")%>',cls:"el-icon el-icon-operation-recommend CODE_CPE_UPGRADE_FILE hidden",code:'recommend',show:recommendFlag},
					{label:'<%=rb.getString("QuXiaoTuiJian")%>',cls:"el-icon el-icon-operation-cancel-recommend CODE_CPE_UPGRADE_FILE hidden",code:'recommend',show:!recommendFlag},
				];
			}
            
            vm.$nextTick(function(){
                document.body.click();
                vm.$refs.menu.show(evt)
            })
        },
		/**
		* 菜单点击事件
		* @param ev{object}   行数据
		*/ 
        menuClick(evt) {
			var vm = this;

			var codes = {
	    		view:this.upgradeFileInfo,  // 详情
	    		download:this.upgradeFileDownload,	// 下载
	    		modify:this.upgradeFileModify,	// 修改
	    		del:this.upgradeFileDel,	// 删除
	    		recommend:this.upgradeFileRecommend,	// 推荐  取消推荐
	    	},
			recommend = '',
			fileType = 9,
			fileName = vm.rowData.file_name;
			if(vm.rowData.recommend == '1'){//说明此文件是推荐文件
				recommend = '0';
			}else{
				recommend = '1';
			}
			if(codes[evt.code]) {
				if(evt.code == 'download'){
					codes[evt.code](fileName,fileType);
				}
				else if(evt.code == 'recommend'){
					codes[evt.code](vm.rowData.id,recommend)
				}else{
					codes[evt.code](vm.rowData.id,vm.rowData);
				} 
			}
        },
		/**
		 * 查看文件
		 * @param id:当前数据id
		*/
		upgradeFileInfo(id){
			var vm = this;
			vm.slideHeader = true;
			vm.slideOpenType = 'fileImport';
			vm.slideUrl = '${ctx}/cell/version/loadPage.action?code='+vm.fileTypeNum;
			vm.slideFooter = false;
			vm.slidePosition = 'top';
			vm.slideHeight = '100%';
			vm.slideWidth = '100%';
            vm.slideSubmitLoading = false;
			vm.slideTitle = '<%=rb.getString("XinXi")%>';
			vm.$refs.upgradeSlide.showSlide(()=>{
				if(vm.fileTypeNum == 'image'){
					eventBus.$emit('cpe-upgrade-init','view',id);
				}else if(vm.fileTypeNum == 'mid'){
					eventBus.$emit('cpe-upgrade-init','view',vm.rowData.version,vm.rowData.modelName);
				}else{
					eventBus.$emit('cpe-upgrade-init','view',vm.rowData.version);
				}
				
			})
			event.stopPropagation();// 禁止事件穿透
		},
		/**
		 * 下载文件
		 * @param fileName:文件名称
		 * @param fileType: 类型
		*/
		upgradeFileDownload(fileName,fileType){
			var vm = this;
			axios.post('${ctx}/omc/version/file/fileIsExist.action',stringify({
	    		fileName : fileName,
	    		fileType:fileType
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
	    			exportByForm("${ctx}/omc/version/file/downLoadFile.action",{
						fileName: fileName,
						fileType: fileType
					});
	    		}else{
	    			vm.$message.error(data["message"]) //错误提示信息
	    		}
	    	}) 
		},
		/**
		 * 修改文件
		 * @param id:当前数据id
		*/
		upgradeFileModify(id){
			var vm = this;
			vm.slideOpenType = 'fileImport';
			vm.slideHeader = true;
			vm.slideUrl = '${ctx}/cell/version/loadPage.action?code='+vm.fileTypeNum;;
			vm.slideFooter = true;
			vm.slidePosition = 'top';
			vm.slideHeight = '100%';
			vm.slideWidth = '100%';
            vm.slideSubmitLoading = false;
			vm.slideTitle = '<%=rb.getString("XiuGai")%>';
			vm.$refs.upgradeSlide.showSlide(()=>{
				if(vm.fileTypeNum == 'image'){
					eventBus.$emit('cpe-upgrade-init','edit',id);
				}else if(vm.fileTypeNum == 'mid'){
					eventBus.$emit('cpe-upgrade-init','edit',vm.rowData.version,vm.rowData.modelName);
				}else{
					eventBus.$emit('cpe-upgrade-init','edit',vm.rowData.version);
				}
			})
		},
		/**
		 * 删除文件
		 * @param id:当前数据id
		*/
		upgradeFileDel(id,rowData){
			var vm = this,
				urls = '',
				params = {};
			if(vm.fileTypeNum == 'image'){
				params.fileID = id;
				urls = '${ctx}/cell/version/deleteVersionFile.action';
			}else if(vm.fileTypeNum == 'mid'){
				params.midVersion = rowData.version;
				params.modelName = rowData.modelName?rowData.modelName:'';
				urls = '${ctx}/cell/version/delMidVersionInfo.action';
			}else{
				params.moduleVersion = rowData.version;
				urls = '${ctx}/cell/version/delModuleVersionInfo.action';
			}
			vm.$confirm('<%=rb.getString("QueDingShanChuWenJian")%>','<%=rb.getString("QueRen")%>',{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		closeOnClickModal:false
	    	}).then(function(){
				axios.post(urls,stringify(params)).then(function(response){
					var data = response.data;
					if(data) {
						if(data["success"]){
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success'
							});
							if(vm.fileTypeNum == 'image'){
								vm.$refs.cpeSystemUpgradeFileTable.refresh();
							}else if(vm.fileTypeNum == 'mid'){
								vm.$refs.middleFileTable.refresh();
							}else{
								vm.$refs.moduleFileTable.refresh();
							}
						}else{
							vm.$message.error(data["message"])
						}
					}
				}).catch(function(error){})
			});
		},
		/**
		 * 推荐
		 * @param id:当前数据id
		 * @param recommend:推荐状态  0：未推荐  1：已推荐
		*/
		upgradeFileRecommend(id,recommend){
			var vm = this,
				params = {
					id: id,
					recommend:recommend
				};
			axios.post('${ctx}/cell/version/updateRecommendStatus.action',stringify(params)).then(function(response){
					var data = response.data;
					if(data) {
						if(data["success"]){
							vm.$refs.cpeSystemUpgradeFileTable.refresh()
						}else{
							vm.$message.error(data["message"])
						}
					}
				}).catch(function(error){})
		},
		// slide 提交
		submitSlide(){
			var vm = this;
			if(vm.slideOpenType == 'fileImport'){
				eventBus.$emit('cpe-upgrade-importSubmit');
			}else{
				eventBus.$emit('cpe-upgrade-addSubmit');
			}
		},
		// 直接关闭slide事件
        hideSlide(){
			var vm = this;
			vm.$refs.upgradeSlide.hide();
			if(vm.activeName == 'file'){
				if(vm.fileTypeNum == 'image'){
					vm.$refs.cpeSystemUpgradeFileTable.refresh();
				}else if(vm.fileTypeNum == 'mid'){
					vm.$refs.middleFileTable.refresh();
				}else{
					vm.$refs.moduleFileTable.refresh();
				}
			}else{
				vm.$refs.cpeUpgradeTaskListTable.refresh();
				vm.$refs.queryResultTable.refresh();
			}
        },
		// 条件关闭slide事件 
		closeSlide(){
			var vm = this;
			if(vm.slideOpenType == 'fileImport'){
				eventBus.$emit('cpe-upgrade-cancelImport');
			}else{
				eventBus.$emit('cpe-upgrade-addCancel');
			}
            
		},
        // 系统升级文件搜索事件
        querySystemUpgradeFile(val){
            var vm = this;
            vm.system_params.searchText = val;
        },
		queryMiddleUpgradeFile(val){
			var vm = this;
            vm.middle_params.version = val;
		},
		queryModuleUpgradeFile(val){
			var vm = this;
            vm.module_params.version = val;
		},
		// 导入文件按钮
        importUpgradeFile(){
			var vm = this;
			vm.slideOpenType = 'fileImport';
			vm.slideHeader = true;
			vm.slideUrl = '${ctx}/cell/version/loadPage.action?code='+vm.fileTypeNum;
			vm.slideFooter = true;
			vm.slidePosition = 'top';
			vm.slideHeight = '100%';
			vm.slideWidth = '100%';
            vm.slideSubmitLoading = false;
			vm.slideTitle = '<%=rb.getString("DaoRuWenJian")%>';
			vm.$refs.upgradeSlide.showSlide(()=>{
				if(vm.fileTypeNum == 'image'){
					eventBus.$emit('cpe-upgrade-init','add','');
				}else if(vm.fileTypeNum == 'mid'){
					eventBus.$emit('cpe-upgrade-init','add','','');
				}else{
					eventBus.$emit('cpe-upgrade-init','add','');
				}
			})
		},
		/**
		* 版本类型转换
		* @param value：表格的数据
		*/
		towhoFormat(value,rowData,rowIndex){
			if(value =='all'){
				return "GA"
			}else if(value == 'beta'){
				return "Beta"
			}else if(value == 'none'){
				return "Test"
			}
		},
		dateChange(val) {
			var vm = this;
			vm.timeRange = val;
			if(val != null){
				this.queryTaskParams.startTime = vm.timeRange[0];
    			this.queryTaskParams.endTime = vm.timeRange[1];
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
			Object.assign(vm.queryDeviceParams, params);
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
			Object.assign(vm.queryDeviceParams, params);
			document.body.click();
		},
	},
	mounted(){
		this.getAdvanceCntent();
		eventBus.$off('hide-cpeUpgrade-slide').$on('hide-cpeUpgrade-slide',this.hideSlide);
		this.init();
	}
	
})

</script> 
