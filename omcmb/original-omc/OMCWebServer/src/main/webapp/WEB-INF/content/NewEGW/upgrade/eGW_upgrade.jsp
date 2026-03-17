<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#egwUpgradePage .el-badge{
	    position:relative;
    }
    #egwUpgradePage .el-badge__content{
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
    #egwUpgradePage .el-icon-star-badge:before{
        color:#F3916C;
    }
	#egwUpgradePage .container .cmenu{
		z-index: 361!important;
	}
	#egwUpgradePage .deviceTable{
		height: calc(50% - 5px);
	}
	#egwUpgradePage .flex-form {
		display: flex;
		flex-wrap: wrap;
	}
	#egwUpgradePage .flex-form .el-form-item {
		margin-right: 100px;
		margin-bottom: 10px;
	}
	#egwUpgradePage .upgradeTaskBox{
		box-sizing: border-box;
		height: calc(50% - 5px);
		background-color: #FFFFFF;
		overflow: auto;
	}
	#egwUpgradePage .upgradeTaskBox .el-tabs{
		height: 100%;
	}
	#egwUpgradePage .upgradeTaskBox .el-tabs__header{
		border-top: none;
		border-bottom: 1px solid #E9E9E9;
	}
	#egwUpgradePage .upgradeTaskBox .el-tabs__item{
		font-size:14px;
	}
	#egwUpgradePage .upgradeTaskBox .el-tabs--top{
		border:none;
	}
	#egwUpgradePage .upgradeTaskBox .el-tabs__nav-scroll{
		margin-left:10px;
	}
	#egwUpgradePage .taskHeadBox{
		position: absolute;
		top:15px;
		right: 0px;
		display: flex;
		z-index: 99;
	}
	#egwUpgradePage .taskHeadBox .statisticsDiv{
		height: 16px;
		line-height: 16px;
		display: flex;
		overflow: hidden;
		font-size: 14px;
	}
	#egwUpgradePage .statisticsDiv > div:first-child{
		padding: 0px 0 0 10px;
		color: #4D84FF;
		background: #FFFFFF;
		height: 16px;
		line-height: 16px;
	}
	#egwUpgradePage .statisticsDiv > div:last-child{
		padding: 0px 10px;
	}
	#egwUpgradePage .statisticsDiv .el-icon, #egwUpgradePage .statisticsSuccessDiv .el-icon, #egwUpgradePage .statisticsFailDiv .el-icon{
		margin-right: 6px;
		font-size: 14px;
	}
	#egwUpgradePage .resultHeadBox{
		position: absolute;
		top:17px;
		right: 60px;
		display: flex;
		z-index: 99;
	}
	#egwUpgradePage .resultHeadBox .statisticsSuccessDiv{
		height: 14px;
		line-height: 14px;
		display: flex;
		font-size: 14px;
		border-right: 1px solid #DFE2EE;
	}
	#egwUpgradePage .statisticsSuccessDiv .el-icon::before{
		color:#67D972;
	}
	#egwUpgradePage .statisticsSuccessDiv > div:first-child{
		padding: 0px 10px;
		color: #67D972;
	}
	#egwUpgradePage .statisticsSuccessDiv > div:last-child{
		padding: 0px 10px 0 0;
	}
	#egwUpgradePage .resultHeadBox .statisticsFailDiv{
		height: 14px;
		line-height: 14px;
		display: flex;
		font-size: 14px;
	}
	#egwUpgradePage .statisticsFailDiv .el-icon::before{
		color: #E88282;
		content:'\e6fb';
	}
	#egwUpgradePage .statisticsFailDiv > div:first-child{
		padding: 0px 10px;
		color: #E88282;
	}
	
	#egwUpgradePage .device_item{
		position:relative;
	}
	#egwUpgradePage .device_item .el-ctable-toolbar {
		padding: 0 !important;
	}
	#egwUpgradePage .list_item .el-tabs__item{
		font-size:14px;
	}
	#egwUpgradePage .list_item .el-tabs--top{
		border:none;
	}
	#egwUpgradePage .list_item{
		position:relative;
	}
	#egwUpgradePage .file_item .queryGroup{
		margin-left:20px;
	}
	#egwUpgradePage .file_item .el-ctable-toolbar {
		padding: 0 !important;
	}
	#egwUpgradePage .el-tabs__header {
		border-bottom: 1px solid #D5DCEC !important;
	}
	#egwUpgradePage .commonTabsTop .el-tabs__header {
		border: 1px solid #D5DCEC;
		border-bottom: 0;
		border-radius: 8px 8px 0 0;
		padding: 0 20px;
	}
	#egwUpgradePage .list_item .el-tabs__header {
		border: 0;
	}
	#egwUpgradePage .upgradeItemBoxCls{
		height: 100%;
		position: relative;
		display: flex;
	}
	#egwUpgradePage .upgradeItemBoxCls >div{
		overflow: hidden;
	}
	#egwUpgradePage .upgradeItemBoxCls .upgradeMainPageBox{
		position: relative;
		flex: 1;
	}
	#egwUpgradePage .upgradeItemBoxCls .importFileBoxCls{
		flex: 0 1 360px;
		margin-left: 10px;
		position: relative;
		background-color: #FFFFFF;
		box-shadow: 0px 0px 10px 1px #E9EDF9;
		border-radius: 10px;
		border: 1px solid #E9EDF9;
		box-sizing: border-box;
	}
	#egwUpgradePage .newTabs .el-ctable-toolbar{
		padding: 0px!important;
	}
	#egwUpgradePage .el-radio.is-bordered,.addDeviceDialog .el-radio.is-bordered{
		height: 30px;
		padding: 7px 20px 0 10px;
	}
	#egwUpgradePage .el-radio.is-bordered+.el-radio.is-bordered,.addDeviceDialog .el-radio.is-bordered+.el-radio.is-bordered{
		margin-left: 15px;
	}

	#egwUpgradePage .rightOutBoxHeadCls{
		height: 50px;
		display: flex;
		align-items: center;
		font-weight: 600;
		font-size: 14px;
		justify-content: space-between;
		padding: 0px 20px;
		border-bottom: 1px solid #E9EDF9;
	}
	#egwUpgradePage .rightItemMainBox{
		padding: 20px;
	}
	#egwUpgradePage .rightItemMainBox .el-input,#egwUpgradePage .rightItemMainBox .el-select{
		width: 100%;
	}
	#egwUpgradePage .importFileBoxCls .footer{
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
	#egwUpgradePage .greyIcon::before{
		color: #7A7992;
		font-size: 14px;
	}
	#egwUpgradePage .labelSlotCls > span{
		color: #999999;
		font-size: 12px;
		margin-left: 10px;
	}
	#egwUpgradePage .el-radio-button:focus:not(.is-focus):not(.is-disabled){
		-webkit-box-shadow: none!important;
		box-shadow: none!important;
	}
	#egwUpgradePage .importFileBoxCls .el-select .el-input.is-disabled .el-input__inner,#egwUpgradePage .importFileBoxCls .el-select .el-input__inner{
		height: unset !important;
	}
</style>
<!-- egw升级 -->
<div class="pageDefault" id='egwUpgradePage' style='border: none;overflow: auto;'>
	<div class="container" style="min-width: 1220px;">
		<div class="upgradeItemBoxCls">
			<div class="upgradeMainPageBox">
				<el-tabs class="fit commonTabsTop" v-model="activeName" style='height:calc(100% - 2px)'>
					<el-tab-pane name="upgrade" label="<%=rb.getString("ShengJi")%>">
						<div class="deviceTable device_item">
							<el-ctable style='border: 1px solid #D5DCEC; border-radius: 0 0 8px 8px; border-top: 0;'
								:url="deviceTableUrl"
								:query-params="queryDeviceParams" 
								ref="upgradedeviceTable" 
								:height="height" 
								@selection-change='deviceSelect'
								:page-size="pageSize" 
								:page-list="pageList" 
								pagination="true">
									<!-- 列表toolbar -->
								<template slot="toolbar">
									<div class='toolbarHeadBtnBoxCls commonQuery' style='margin-bottom: 0;height:45px;'>
										<div v-if="optBtnShow" class="newIconBoxCls-bt" style="right:20px;top:10px;" @click="addUpgradeTask" tip="<%=rb.getString("ShengJi")%>">
											<span class='el-icon el-icon-circle-upgrade'></span>
										</div>                      		
										<el-query @query="queryDevice" type="normal"  placeholder="<%=rb.getString("eGWBianMa")%> / <%=rb.getString("EGWIP")%>"></el-query>															
									</div>
								</template>
									<!-- 列表columns -->
								<el-table-column v-if="optBtnShow" type="selection" width="45"></el-table-column>
								<el-table-column prop="connectionStatus" width="50">
									<template slot-scope="scope">
										<div :class="{
											'el-icon el-icon-status-conn-off':scope.row.connectionStatus!='Exception' && scope.row.connectionStatus!='On' && scope.row.connectionStatus!='updating' && scope.row.connectionStatus!=1,
											'':scope.row.have_connected==2,
											'conn_exc':scope.row.connectionStatus=='Exception',
											'el-icon el-icon-status-conn-on':scope.row.connectionStatus=='On'||scope.row.connectionStatus=='updating'||scope.row.connectionStatus==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
									</template>
								</el-table-column>
								<el-table-column prop="egwSn" label="<%=rb.getString("eGWBianMa")%>" min-width="150"></el-table-column>
								<el-table-column prop="egwName" label="<%=rb.getString("EGWMingCheng")%>" min-width="150"></el-table-column>
								<el-table-column prop="egwIp" label="<%=rb.getString("EGWIP")%>" min-width="120"></el-table-column>
								<el-table-column prop="egwPort" label="<%=rb.getString("EGWDuanKou")%>" min-width="140"></el-table-column>
								<el-table-column prop="softwareVersion" label="<%=rb.getString("BanBen")%>" min-width="150"></el-table-column>
							</el-ctable>
						</div>
						<div class="upgradeTaskBox list_item" style='border: 1px solid #D5DCEC; border-radius: 8px; margin-top:10px;'>
							<el-tabs v-model="upgrade_activeName" class='newTabs'>
								<el-tab-pane label="<%=rb.getString("RenWuLieBiao")%>" name="task">
									<div class="taskHeadBox">
										<div class="statisticsDiv">
											<div><span class="el-icon el-icon-status-waiting1"></span><%=rb.getString("DengDai")%></div>
											<div>{{statisticTaskStatus.waitingNum}}</div>
										</div>
										<div class="statisticsDiv">
											<div><span class="el-icon el-icon-status-inProgress"></span><%=rb.getString("JinXingZhong")%></div>
											<div>{{statisticTaskStatus.processingNum}}</div>
										</div>
										<div class="statisticsDiv">
											<div><span class="el-icon el-icon-status-suspend"></span><%=rb.getString("ZanTing")%></div>
											<div>{{statisticTaskStatus.suspendedNum}}</div>
										</div>
										<div class="statisticsDiv">
											<div><span class="el-icon el-icon-status-terminate"></span><%=rb.getString("JieShu")%></div>
											<div>{{statisticTaskStatus.endNum}}</div>
										</div>
									</div>
									<el-ctable id="egwUpgradeTaskTable" time=6  @load-success="taskTableLoadSuccess" ref="egwUpgradeTaskListTable" :url="taskListTableUrl" :query-params="queryTaskParams" :page-size="20" :pagination=true height="100%">
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
										<el-table-column prop="taskId"  v-if="false"></el-table-column>
										<el-table-column prop="taskName" show-overflow-tooltip label="<%=rb.getString("RenWuMingCheng")%>"  min-width="200"></el-table-column>
										<el-table-column prop="createUser" label="<%=rb.getString("CaoZuoRen")%>"  min-width="100" ></el-table-column>
										<el-table-column prop="createTime" label="<%=rb.getString("CaoZuoShiJian")%>" min-width="180"></el-table-column>
										<el-table-column prop="fileName" show-overflow-tooltip label="<%=rb.getString("WenJianMing")%>" min-width="200"></el-table-column>
										<el-table-column prop="version" label="<%=rb.getString("BanBen")%>" min-width="200"></el-table-column>
										<el-table-column prop="productType" label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" min-width="130"></el-table-column>
										<el-table-column prop="taskStatus" label="<%=rb.getString("ZhuangTai")%>" min-width="120">
											<template slot-scope="scope">
												<div v-html="taskTableStatus(scope.row.taskStatus)"></div>
											</template>
										</el-table-column>
										<el-table-column prop="taskProgress" label="<%=rb.getString("JinDu")%>" min-width="100"></el-table-column>
										<el-table-column prop="taskResult" label="<%=rb.getString("JieGuo")%>" min-width="100" :formatter="taskTableResult"></el-table-column>
										<el-table-column prop="startTime" label="<%=rb.getString("KaiShiShiJian")%>" min-width="180"></el-table-column>
										<el-table-column prop="endTime" label="<%=rb.getString("JieShuShiJian")%>" min-width="180"></el-table-column>
									</el-ctable>
									<el-cmenu ref="taskMenu" :data="taskMenus" @click="taskMenuClick"></el-cmenu>
								</el-tab-pane>
								<el-tab-pane label="<%=rb.getString("SheBeiLieBiao")%>" name="device"> 
									<div class="resultHeadBox">
										<div class="statisticsSuccessDiv">
											<div><span class="el-icon el-icon-circle-success"></span><%=rb.getString("ChengGong")%></div>
											<div>{{statisticDeviceResult.sucNum}}</div>
										</div>
										<div class="statisticsFailDiv">
											<div><span class="el-icon el-icon-circle-close"></span><%=rb.getString("ShiBai")%></div>
											<div>{{statisticDeviceResult.failNum}}</div>
										</div>
									</div>
									<!--:url="queryResultUrl"  :data-->
									<el-ctable 
										id="egwUpgradeResultTable"
										ref="queryResultTable" 
										:url="queryResultUrl" 
										:query-params="queryResultParams"
										@load-success="deviceTableLoadSuccess"
										height="100%"
										time=6
										:page-size="20" 
										:pagination=true>
										
										<template slot="toolbar">
											<div class='toolbarHeadBtnBoxCls commonQuery' style='margin-bottom: 0;height:45px; position: relative;'>
												<div class="newIconBoxCls-bt" style="right:10px;top:10px;" @click="exportResultTable" tip="<%=rb.getString("DaoChu")%>">
													<span class='el-icon el-icon-operation-export'></span>
												</div>
												<el-query type="normal" @query="queryEgwUpgradeResult" placeholder="<%=rb.getString("eGWBianMa")%> / <%=rb.getString("EGWIP")%>"></el-query>
											</div>
										</template>
										<el-table-column prop="ID" v-if="false"></el-table-column>
										<el-table-column prop="egwSn" label="<%=rb.getString("eGWBianMa")%>" min-width="150"></el-table-column>
										<el-table-column prop="egwName" label="<%=rb.getString("EGWMingCheng")%>" min-width="150"></el-table-column>
										<el-table-column prop="egwIp" label="<%=rb.getString("EGWIP")%>" min-width="120"></el-table-column>
										<el-table-column prop="taskName" show-overflow-tooltip label="<%=rb.getString("RenWuMingCheng")%>"  min-width="280"></el-table-column>
										<el-table-column prop="originalVersion" show-overflow-tooltip label="<%=rb.getString("ChuShiBanBen")%>" min-width="200"></el-table-column>
										<el-table-column prop="productType" show-overflow-tooltip label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" min-width="120"></el-table-column>
										<el-table-column prop="progressStatus" label="<%=rb.getString("ZhuangTai")%>" min-width="120">
											<template slot-scope="scope">
												<div v-html="resultTableStatus(scope.row.progressStatus)"></div>
											</template>
										</el-table-column>
										<el-table-column prop="progressResult" label="<%=rb.getString("JieGuo")%>" min-width="110" :formatter="resultTableResult"></el-table-column>
										<el-table-column prop="failureReason" show-overflow-tooltip label="<%=rb.getString("PCILOCKShiBaiYuanYin")%>" min-width="120"></el-table-column>
										<el-table-column prop="startTime" label='<%=rb.getString("KaiShiShiJian")%>' min-width="160" ></el-table-column>
										<el-table-column prop="endTime" label='<%=rb.getString("JieShuShiJian")%>'  min-width="160"></el-table-column>
									</el-ctable>
								</el-tab-pane>
							</el-tabs>
						</div>
					</el-tab-pane>
					<el-tab-pane name="file" label="<%=rb.getString("WenJian")%>" class='file_item'>
						<!-- 升级文件 表格组件 :url="fileTableUrl" -->
						<el-ctable  style='border: 1px solid #D5DCEC;border-top:none; border-radius: 0 0 8px 8px; height: calc(100% - 2px);'
							:url="fileTableUrl"
							:query-params="file_params" 
							ref="upgradeFileTable" 
							:height="height" 
							:page-size="pageSize" 
							:page-list="pageList" 
							pagination="true">
								<!-- 列表toolbar -->
							<template slot="toolbar">
								<div class='toolbarHeadBtnBoxCls commonQuery' style='margin-bottom: 0; border-top: 0;height:45px;'>
									<div class="newIconBoxCls-bt CODE_EGW hidden" style="right:20px;top:10px;" @click="importFileClick" tip="<%=rb.getString("DaoRuWenJian")%>">
										<span class='el-icon el-icon-operation-import'></span>
									</div> 
									<el-query type="normal" @query="queryUpgradeFile" placeholder="<%=rb.getString("BanBen")%>"></el-query>
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
							<el-table-column prop="product_type" label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" min-width="150"></el-table-column>
							<el-table-column prop="file_size" label="<%=rb.getString("WenJianDaXiao")%>" min-width="120"></el-table-column>
							<el-table-column prop="upload_time" label="<%=rb.getString("ShangChuanShiJian")%>" min-width="140"></el-table-column>
						
						</el-ctable>
					</el-tab-pane>
					<!-- 菜单 -->
					<el-cmenu ref="menu" @click="menuClick" :data="menus"></el-cmenu>
				</el-tabs>
			</div>
			<div class="importFileBoxCls" v-show="importFileShow">
				<div class="rightOutBoxHeadCls">
					<span>{{rightOutBoxTitle}}</span>
					<span class="el-icon el-icon-close greyIcon" @click="rightBoxClose"></span>
				</div>
				<div class="rightItemMainBox">
					<el-form ref="importFileForm" :model="importFileForm" label-position="top" :rules="importFileFormRules" :hide-required-asterisk=true>
						<el-form-item prop="productType" label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" >
							<el-input v-model='importFileForm.productType' :disabled="true"></el-input>
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
						<!--<el-form-item label='<%=rb.getString("TuiJian")%>' prop='recommend'>
							<el-select v-model='importFileForm.recommend' :disabled="importViewFlag">
								<el-option label='<%=rb.getString("Shi")%>' value='1'></el-option>
								<el-option label='<%=rb.getString("Fou")%>' value='0'></el-option>
							</el-select>
						</el-form-item>-->
						<el-form-item label="<%=rb.getString("MiaoShu")%>" prop='description'>
							<el-input v-model='importFileForm.description' type='textarea' :rows='2' :disabled="importViewFlag"></el-input>
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
		<!-- slide -->
		<el-slide ref="upgradeSlide" id="upgradeSlide" :url='slideUrl' :title="slideTitle" :footer="slideFooter" :header="slideHeader" :position="slidePosition"
			:height="slideHeight"  :width='slideWidth' :subloading="slideSubmitLoading" @ok="submitSlide"  @cancel="closeSlide" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
		</el-slide>
		
    </div>
</div>
<script type="text/javascript">
var egwUpgrade = new Vue({
	el:'#egwUpgradePage',
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
		return {
			activeName:'upgrade',
            file_params:{
                timeZone:timeZone,
                searchText:'',
            },
            fileTableUrl:'${ctx}/egw/softwareFile/querySoftwareFilePageList.action',

			deviceTableUrl:"${ctx}/egw/monitor/getEgwMonitorPageList.action",
			productValue: "1", //产品类型
			selection:[],
			
			queryDeviceParams:{
				search_text:'',
				timeZone:timeZone,
			},
			
			taskListTableUrl:'${ctx}/egw/softwareUpgrade/getTaskList.action',
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
			queryResultUrl:'${ctx}/egw/softwareUpgrade/getTaskDevices.action',
			queryResultParams:{
				timeZone: timeZone,
				searchText:'',
			},
			deviceGroups:[],//高级查询设备组选择下拉内容
			versions:[],//高级查询版本选择下拉内容
			modelName:[], // 高级查询model下拉内容
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
            rowDataFile:'',
			taskRowData:'',
			statisticTaskStatus:{
				waitingNum: 0,
				processingNum: 0,
				suspendedNum: 0,
				endNum: 0
			},
			statisticDeviceResult:{
				sucNum: 0,
				failNum: 0
			},
			slideOpenType:'',

			importFileShow:false,
			importFileType:'',
			importViewFlag:false,
			importFileForm:{
				file:'',
				fileName:'',
				productType:'WCG',
				fileName:'',
				version:'',
				// recommend:'1',
				description:'',
			},
			importFileFormRules:{
				fileName:[
					{validator: fileNameValidate}
				],
				version:[
					{validator: versionValidate,trigger:'blur'}
				],
				productType:[
					{required:true,message:'<%=rb.getString("ShuRuBiTianXiang")%>',trigger:'change'}
				],
			},
			fileTypeTip:'<%=rb.getString("RPMWenJianTiShi")%>',
			fileErrorData:'',
			difFileObj:{
				upgrade:{
					fileTip:'<%=rb.getString("RPMWenJianTiShi")%>',
					fileFmt:'rpm',
					fileErrorMsg:'<%=rb.getString("ZhIZhiChiRPMWenJian")%>',
				}
			},
            multipartMaxFileSize: multipartMaxFileSize
		}
	},
    computed:{
		isSuperAdmin() {
			return is_super_user == 'true';
		},
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
			return writableMap['CODE_EGW'] == true;
		},
    },
	watch:{
		timeRange(newVal){
			var vm = this;
			if(!newVal){
				newVal = [];
				vm.queryTaskParams.startTime = '';
    			vm.queryTaskParams.endTime = '';
			}
		},
		"importFileForm.fileName":function(val){
			var vm = this;
			if(vm.importFileType == 'add'){
				this.getVersion(val)
			}
		},
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
			vm.slideUrl = '${ctx}/egw/pageForward/goEGWUpgradeAddTask.action?timeZone='+ timeZone;
			vm.slideFooter = true;
			vm.slidePosition = 'top';
			vm.slideHeight = '100%';
			vm.slideWidth = '100%';
            vm.slideSubmitLoading = false;
			vm.slideTitle = '<%=rb.getString("XinJianShengJiRenWu")%>';
			vm.$refs.upgradeSlide.showSlide(()=>{
				eventBus.$emit('egw-upgrade-taskInit','','addTask');
			})
		},
		// 设备表格选择事件
		deviceSelect(selection){
			var vm = this;

			vm.selection = selection
		},
		// egw 升级任务结果 导出
		exportResultTable(){
			var vm = this;

			exportByForm("${ctx}/egw/softwareUpgrade/exportTaskDevices.action",{
				timeZone: timeZone,
				searchText: vm.queryResultParams.searchText
			});
		},
		// egw 升级任务结果 表格模糊查询
		queryEgwUpgradeResult(val){
			var vm = this;
			vm.queryResultParams.searchText= val;
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
		// 设备表格 模糊查询
		queryDevice(val){
			var vm = this;
			vm.queryDeviceParams.search_text= val;
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
		// 升级任务表格 打开操作菜单
		taskOptClick(row,evt){
			var vm = this;

			vm.taskRowData = row;
			var status = row.taskStatus;
			vm.taskMenus = [
                {label:'<%=rb.getString("KaiShi")%>',cls:"el-icon el-icon-operation-start CODE_EGW hidden",code:'start'},
                {label:'<%=rb.getString("ZanTing")%>',cls:"el-icon el-icon-operation-awaiting CODE_EGW hidden",code:'wait'},
				{label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'info'},
                {label:'<%=rb.getString("ZhongZhi")%>',cls:"el-icon el-icon-operation-terminate CODE_EGW hidden",code:'end'},
				// {label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit CODE_EGW hidden",code:'mod'},
                {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete CODE_EGW hidden",code:'del'},
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
				start:this.egwUpgradeTaskStart,	// 开始
				wait:this.egwUpgradeTaskStop,	// 暂停
				end:this.egwUpgradeTaskTerminate,	// 终止
	    		info:this.egwUpgradeTaskInfo,  // 详情
	    		mod:this.egwUpgradeTaskModify,	// 修改
	    		del:this.egwUpgradeTaskDel,	// 删除
	    	};
			if(codes[evt.code]){
				codes[evt.code](vm.taskRowData.taskId);
			}
		},
		/**
		 * 激活任务
		 * @param id:数据id
		 * @param fileType: 类型
		*/
		egwUpgradeTaskStart(id){
			var vm = this;
			axios.post('${ctx}/egw/softwareUpgrade/activeTask.action',stringify({
	    		taskId : id,
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
					vm.$refs.egwUpgradeTaskListTable.refresh()
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
		egwUpgradeTaskStop(id){
			var vm = this;
			axios.post('${ctx}/egw/softwareUpgrade/suspendTask.action',stringify({
	    		taskId : id,
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
					vm.$refs.egwUpgradeTaskListTable.refresh()
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
		egwUpgradeTaskTerminate(id){
			var vm = this;
			axios.post('${ctx}/egw/softwareUpgrade/terminateTask.action',stringify({
	    		taskId : id,
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
					vm.$refs.egwUpgradeTaskListTable.refresh()
	    		}else{
	    			vm.$message.error(data["message"]) //错误提示信息
	    		}
	    	}) 
		},
		/**
		 * 查看任务
		 * @param id:当前数据id
		*/
		egwUpgradeTaskInfo(id){
			var vm = this;
			vm.slideHeader = true;
			vm.slideOpenType = 'upgradeTask';
			vm.slideUrl = '${ctx}/egw/pageForward/goEGWUpgradeAddTask.action?timeZone='+ timeZone;
			vm.slideFooter = false;
			vm.slidePosition = 'top';
			vm.slideHeight = '100%';
			vm.slideWidth = '100%';
            vm.slideSubmitLoading = false;
			vm.slideTitle = '<%=rb.getString("XinXi")%>';
			vm.$refs.upgradeSlide.showSlide(()=>{
				eventBus.$emit('egw-upgrade-taskInit',id,'taskView');
			})
		},
		/**
		 * 修改任务
		 * @param id:当前数据id
		*/
		egwUpgradeTaskModify(id){
			var vm = this;
			vm.slideOpenType = 'upgradeTask';
			vm.slideHeader = true;
			vm.slideUrl = '${ctx}/egw/pageForward/goEGWUpgradeAddTask.action.action?timeZone='+ timeZone;
			vm.slideFooter = true;
			vm.slidePosition = 'top';
			vm.slideHeight = '100%';
			vm.slideWidth = '100%';
            vm.slideSubmitLoading = false;
			vm.slideTitle = '<%=rb.getString("XiuGai")%>';
			vm.$refs.upgradeSlide.showSlide(()=>{
				eventBus.$emit('egw-upgrade-taskInit',id,'taskEdit',vm.productValue);
			})
		},
		/**
		 * 删除升级任务
		 * @param id:当前数据id
		*/
		egwUpgradeTaskDel(id){
			var vm = this,
				params = {
					taskId: id,
				};
			vm.$confirm('<%=rb.getString("QueRenShanChuRenWu")%>','<%=rb.getString("QueRen")%>',{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		closeOnClickModal:false
	    	}).then(function(){
				axios.post('${ctx}/egw/softwareUpgrade/delTask.action',stringify(params)).then(function(response){
					var data = response.data;
					if(data) {
						if(data["success"]){
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success'
							});
							vm.$refs.egwUpgradeTaskListTable.refresh()
						}else{
							vm.$message.error(data["message"])
						}
					}
				}).catch(function(error){})
			});
		},
		// 菜单关闭
        hideMenus() {
            this.$refs.menu.hide();
			this.$refs.taskMenu.hide();
        },
        // 打开操作菜单
        optClick(row,evt) {
            var vm = this,recommendFlag="";

            vm.rowDataFile = row;
			if(row.recommend == '1'){//说明此文件是推荐文件
				recommendFlag = false;
			}else{
				recommendFlag = true;
			}
			
			vm.menus = [
				{label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'view'},
				{label:'<%=rb.getString("XiaZai")%>',cls:"el-icon el-icon-operation-download CODE_EGW hidden",code:'download'},
				{label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit CODE_EGW hidden",code:'modify'},
				{label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete CODE_EGW hidden",code:'del'},
				// {label:'<%=rb.getString("TuiJian")%>',cls:"el-icon el-icon-operation-recommend CODE_EGW hidden",code:'recommend',show:recommendFlag},
				// {label:'<%=rb.getString("QuXiaoTuiJian")%>',cls:"el-icon el-icon-operation-cancel-recommend CODE_EGW hidden",code:'recommend',show:!recommendFlag},
			];
			
            
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
				fileName = vm.rowDataFile.file_name,
				id = vm.rowDataFile.id;
			if(vm.rowDataFile.recommend == '1'){//说明此文件是推荐文件
				recommend = '0';
			}else{
				recommend = '1';
			}
			if(codes[evt.code]) {
				if(evt.code == 'download'){
					codes[evt.code](fileName,id);
				}
				else if(evt.code == 'recommend'){
					codes[evt.code](id,recommend)
				}else{
					codes[evt.code](vm.rowDataFile.id,vm.rowDataFile);
				} 
			}
        },
		/**
		 * 查看文件
		 * @param id:当前数据id
		*/
		upgradeFileInfo(id,row){
			var vm = this;
			vm.importFileType = 'view';
			vm.importViewFlag = true;
			Object.keys(vm.importFileForm).forEach(function(key){
				if(key == 'fileName'){
					vm.importFileForm[key] = row.file_name ? row.file_name : '';
				}else if(key == 'productType'){
					vm.importFileForm[key] = row.product_type ? row.product_type : '';
				}else{
					if(row[key] != undefined && row[key] != null){
						vm.importFileForm[key] = row[key];
					}
				}
			});
			vm.importFileShow = true;
			event.stopPropagation();// 禁止事件穿透
		},
		/**
		 * 下载文件
		 * @param fileName:文件名称
		 * @param fileType: 类型
		*/
		upgradeFileDownload(fileName,idVal){
			var vm = this;
			exportByForm("${ctx}/egw/softwareFile/downloadSoftwareFile.action",{
				id: idVal,
				fileName: fileName
			})
		},
		/**
		 * 修改文件
		 * @param id:当前数据id
		*/
		upgradeFileModify(id,row){
			var vm = this;

			vm.importFileType = 'edit';
			vm.importViewFlag = false;
			Object.keys(vm.importFileForm).forEach(function(key){
				if(key == 'fileName'){
					vm.importFileForm[key] = row.file_name ? row.file_name : '';
				}else if(key == 'productType'){
					vm.importFileForm[key] = row.product_type ? row.product_type : '';
				}else{
					if(row[key] != undefined && row[key] != null){
						vm.importFileForm[key] = row[key];
					}
				}
			});
			vm.importFileShow = true;
		},
		/**
		 * 删除文件
		 * @param id:当前数据id
		*/
		upgradeFileDel(id,row){
			var vm = this,
				urls = '',
				params = {};
			params.id = id;
			urls = '${ctx}/egw/softwareFile/delSoftwareFileInfos.action';
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
							vm.$refs.upgradeFileTable.refresh();
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
		// upgradeFileRecommend(id,recommend){
		// 	var vm = this,
		// 		params = {
		// 			id: id,
		// 			recommend:recommend
		// 		};
		// 	axios.post('${ctx}/cell/version/updateRecommendStatus.action',stringify(params)).then(function(response){
		// 			var data = response.data;
		// 			if(data) {
		// 				if(data["success"]){
		// 					vm.$refs.upgradeFileTable.refresh()
		// 				}else{
		// 					vm.$message.error(data["message"])
		// 				}
		// 			}
		// 		}).catch(function(error){})
		// },
		// slide 提交
		submitSlide(){
			var vm = this;
			if(vm.slideOpenType == 'fileImport'){
				eventBus.$emit('egw-upgrade-importSubmit');
			}else{
				eventBus.$emit('egw-upgrade-addSubmit');
			}
		},
		// 直接关闭slide事件
        hideSlide(){
			var vm = this;
			vm.$refs.upgradeSlide.hide();
			if(vm.activeName == 'file'){
				vm.$refs.upgradeFileTable.refresh();
			}else{
				vm.$refs.egwUpgradeTaskListTable.refresh();
				vm.$refs.queryResultTable.refresh();
			}
        },
		// 条件关闭slide事件 
		closeSlide(){
			var vm = this;
			if(vm.slideOpenType == 'fileImport'){
				eventBus.$emit('egw-upgrade-cancelImport');
			}else{
				eventBus.$emit('egw-upgrade-addCancel');
			}
            
		},
        // 系统升级文件搜索事件
        queryUpgradeFile(val){
            var vm = this;
            vm.file_params.searchText = val;
        },
		// 导入文件按钮
        importFileClick(){
			var vm = this;

			vm.importFileType = 'add';
			vm.importViewFlag = false;
			vm.$refs.importFileForm.resetFields();
			vm.importFileShow = true;
		},
		dateChange(val) {
			var vm = this;
			vm.timeRange = val;
			if(val != null){
				this.queryTaskParams.startTime = vm.timeRange[0];
    			this.queryTaskParams.endTime = vm.timeRange[1];
			}
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
			fd.append('fileSize',fileSize);//文件大小
			fd.append('description',vm.importFileForm.description);//描述
			fd.append('productType','eGW');
			fd.append('version',vm.importFileForm.version);
			// fd.append('recommend',vm.importFileForm.recommend);
			vm.fileErrorData = vm.$refs.importFile.uploadFiles[0];
			$('#progressUploadFile').progressbar('setValue', 0);// 将进度条进度置为0
			$("#winUploadPro").window("open");// 打开进度条窗口
			axios.post("${ctx}/egw/softwareFile/uploadSoftwareFile.action",fd,config).then(function(response){
				var data = response.data
				$("#winUploadPro").window("close");// 关闭进度条窗口
				if(data["success"]){
					vm.fileErrorData = '';
					$.messager.alert('<%=rb.getString("TiShi")%>','<%=rb.getString("ShangChuanChengGong")%><%=rb.getString("DouHao")%><%=rb.getString("WenJianMD5Zhi")%><%=rb.getString("MaoHao")%>'+data["message"]);
					vm.$refs.upgradeFileTable.refresh();
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
					id:vm.rowDataFile.id,
					fileName:vm.importFileForm.fileName,
					productType:'eGW',
					version:vm.importFileForm.version,
					// recommend:vm.importFileForm.recommend,
					description:vm.importFileForm.description,
				};
			axios.post("${ctx}/egw/softwareFile/editSoftwareFileInfos.action",stringify(params)).then(function(response){
				var data = response.data;
				if(data) {
					if(data["success"]){
						vm.$message({
							message: '<%=rb.getString("ChengGong")%>',
							type:'success'
						});
						vm.$refs.upgradeFileTable.refresh();
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
					productType:'WCG',
					file:'',
					fileName:'',
					version:'',
					recommend:'1',
					description:'',
				};
			vm.importFileShow = false;
			Object.assign(vm.importFileForm,params)

		},
	},
	mounted(){
		eventBus.$off('hide-egwUpgrade-slide').$on('hide-egwUpgrade-slide',this.hideSlide);
		this.init();
	}
	
})

</script> 
