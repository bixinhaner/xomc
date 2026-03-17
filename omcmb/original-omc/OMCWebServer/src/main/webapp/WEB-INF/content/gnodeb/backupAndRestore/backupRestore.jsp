<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<style>
	#gnbBackupRstore .gnbDeviceWarp,
	#gnbBackupRstore .taskListWarp {
		flex:1;
		overflow:auto;
	}
	#gnbBackupRstore .el-checkbox__label{
		font-size:12px;
	}

	#gnbBackupRstore .filter-opt{
		width:20px;
		height:20px;
	}
	#gnbBackupRstore .dataStatistics{
		position:absolute;
		top:52px;
		right:10px;
		height:16px;
		z-index:99;
		display:flex;
	}
	#gnbBackupRstore .dataStatistics .grantSuspended{
		margin:0 10px;
	}
	#gnbBackupRstore .dataStatistics .enableStatistics{
		display:flex;
		height:16px;
		line-height:16px;
		border-right: 1px solid #D5DCEC;
	}
	#gnbBackupRstore .dataStatistics .enableStatistics .enableTitleStatistics{
		color:#4D84FF;
	}
	#gnbBackupRstore .dataStatistics .enableStatistics .enableNumStatistics{
		color: rgba(0, 0, 0, 0.8);
	}
	#gnbBackupRstore .dataStatistics .enableStatistics .titleCommon{
		padding:0 10px 0 6px;
		font-size:14px;
	}

	#gnbBackupRstore .statusTip{
		margin-right:6px;
	}
	#gnbBackupRstore .deviceStatusTip{
		margin-right:8px;
	}
	#gnbBackupRstore .dataStatistics .statusTip:before,
	#gnbBackupRstore .dataStatistics .deviceStatusTip:before{
		font-size:16px;
	}
	#gnbBackupRstore .el-icon-circle-success:before{
		color:#67D972;
	}
	#gnbBackupRstore .dataStatistics .el-icon-circle-close:before{
		color:#E88282;
		content:'\e6fb';
	}

	#gnbBackupRstore .w270{
		width:400px;
	}

	#gnbBackupRstore .backupRestoreImport .el-form-item{
		margin-bottom:22px;
	}
	#gnbBackupRstore .backupRestoreImport .el-dialog__body{
		height:450px;
		padding:20px 30px 0;
	}
	#gnbBackupRstore .backupRestoreImport .el-dialog__body .importSelect .el-form-item__label{
		margin-top:-7px;
	}
	#gnbBackupRstore .backupRestoreImport .el-dialog__body .el-input__suffix{
		top:4px;
	}
	#gnbBackupRstore .backupRestoreImport .el-dialog__body .el-form-item__error{
		padding-top:0;
	}
	#gnbBackupRstore .backupRestoreImport .el-dialog__footer{
		text-align:left;
		padding:30px 30px;
	}
	#gnbBackupRstore .backupRestoreImport .el-dialog__body thead tr th:first-child .cell{
		display:none;
	}
	#gnbBackupRstore .advanceQuery .el-input__inner{
		color:#333333;
	}
	#gnbBackupRstore .selectDeviceTable .queryGroup{
		margin-left:10px;
	}
	#gnbBackupRstore .queryGroup .el-icon-common-search{
		margin-top:-3px;
	}

	#gnbBackupRstore .queryGroup,
	#gnbBackupRstore .pairgrid-query .el-input__inner{
		height:24px;
	}
	/*归属公共样式  */
	
	#gnbBackupRstore .column-flex .el-checkbox{
		display: block;
		margin-left: 0px;
		padding-top:10px;
	}
	/*
	#gnbBackupRstore .el-button--primary:focus{
		background-color:#4D84FF;
		border-color:#4D84FF;
	}
	*/
	#gnbBackupRstore .warningConfirm .el-button--primary{
		background-color:#FFFFFF !important;
		color:#666666 !important;
		border-color:#DCDFE6 !important;
	}

	#gnbBackupRstore .el-tabs--card>.el-tabs__header .el-tabs__nav{
		margin-left:10px;
		border:none;
	}
	#gnbBackupRstore input::-webkit-input-placeholder{
		color:#CCCCCC !important;
	}
	#gnbBackupRstore input::-moz-input-placeholder{
		color:#CCCCCC !important;
	}
	#gnbBackupRstore input::-ms-input-placeholder{
		color:#CCCCCC !important;
	}
	#gnbBackupRstore .circleIcon{
		min-width:26px;
	}
	#gnbBackupRstore .el-message--info{
		border-color:#81A8FF;
		background-color:#F2F6FF;
	}
	#gnbBackupRstore .el-ctable thead tr th{
		color:#333333;
	}
	#gnbBackupRstore .el-ctable tbody tr td{
		color:#666666;
	}
	#gnbBackupRstore .el-table .descending .sort-caret.descending{
		border-top-color:#4D84FF;
	}
	#gnbBackupRstore .el-dialog__header .el-dialog__headerbtn{
		right:20px;
		top:13px;
	}
	#gnbBackupRstore .el-message__icon{
		margin-right:10px;
	}
	#gnbBackupRstore .importFileWarp{
		height:24px;
		line-height:24px;
		padding:0 10px;
		margin-top:2px;
		border:1px solid #4D84FF;
		border-radius:4px;
		display:flex;
		margin-left:24px;
		cursor:pointer;
		background:#F2F6FF;
	}
	#gnbBackupRstore .importFileWarp span:first-child{
		font-size:16px;
		margin-top:4px;
		margin-right:5px;
	}
	#gnbBackupRstore .importFileWarp span:last-child{
		font-size:12px;
	}
	#gnbBackupRstore .moreFileSelected{
		padding-top:12px;
		display: block;
	}
	#gnbBackupRstore .singleFileSelected{
		padding-top:12px;
		display: block;
		margin-left: 0;
	}
	#gnbBackupRstore .oneFileMoreDevicesTip{
		text-align:left;
		margin-left:10px;
		color: rgba(0, 0, 0, 0.32);
	}

	#gnbBackupRstore .fileImportTypeTip{
		color: rgba(0, 0, 0, 0.32);
		margin-left:10px;
	}
	#gnbBackupRstore .el-radio__inner:hover{
		border-color:#4D84FF;
	}
	#gnbBackupRstore .el-date-editor .el-range__close-icon {
		line-height:20px;
	}
	#profileAddDiv .el-icon-not-installed:before {
		color: #A0B2C8;
	}
	#profileAddDiv .el-icon-status-timeOut:before { 
		color: #19D5F3; 
	}
    #gnbBackupRstore .newTabs .el-ctable-toolbar{
        padding: 0px!important;
    }
</style>
<div class="overflow-cls">
<!-- gNB备份与恢复 -->
<div id="gnbBackupRstore" style="min-width: 1260px; width: 100%; height: 100%;">
	<div class='panelDefault' style="background:unset; border: 0; display:flex; flex-direction:column;">
		<div class="gnbDeviceWarp commonWarp">
			<div>
				<div v-if="gnbIsSuperAdmin">
					<div @click="gnbBackupTaskBtn" class="circleIcon placeholder-bt CODE_GNB_BACKUP_RESTORE hidden" style="right: 100px;top:6px;" placeholder="<%=rb.getString("BeiFen")%>">
						<span class="el-icon el-icon-circle-backup"></span>
					</div>
					<div id='profileAddDiv' @click="gnbPeriodBackupBtn" class="circleIcon placeholder-bt" :class="{'is-disabled': gnbPeriodDataLoading}" style="right: 60px;top:6px;cursor:pointer" :style="{cursor: gnbPeriodDataLoading ? 'not-allowed' : 'pointer', opacity: gnbPeriodDataLoading ? 0.6 : 1}" placeholder="<%=rb.getString("ZhouQiBeiFens")%>">
						<span class="el-icon el-icon-circle-backup"></span>
						<span v-if='gnbPeriodSwitch == "0"' class='el-icon el-icon-not-installed' style='position: absolute; top: 7px; right: 0; font-size: 14px;'></span>
						<span v-if='gnbPeriodSwitch == "1"' class='el-icon el-icon-status-timeOut' style='position: absolute; top: 7px; right: 0; font-size: 14px; '></span>
					</div>

					<div id='profileViewDiv' @click="gnbRestoreBtn" class="circleIcon placeholder-bt CODE_GNB_BACKUP_RESTORE hidden" style="top:6px;right:20px;" placeholder="<%=rb.getString("HuiFu")%>">
						<span class="el-icon el-icon-circle-restore"></span>
					</div>
				</div>
				<div v-else>
					<div @click="gnbBackupTaskBtn" class="circleIcon placeholder-bt CODE_GNB_BACKUP_RESTORE hidden" style="right: 60px;top:6px;" placeholder="<%=rb.getString("BeiFen")%>">
						<span class="el-icon el-icon-circle-backup"></span>
					</div>
					<div id='profileViewDiv' @click="gnbRestoreBtn" class="circleIcon placeholder-bt CODE_GNB_BACKUP_RESTORE hidden" style="top:6px;right:20px;" placeholder="<%=rb.getString("HuiFu")%>">
						<span class="el-icon el-icon-circle-restore"></span>
					</div>
				</div>
			</div>

			<el-ctable ref="gNBDeviceTable"
				id="gNBDeviceTable"
				row-key="small_cell_code"
				:limit="100"
				:time="6"
				:url="gnbDeviceUrl"
				:height="height"
				:query-params="gNBDeviceParams"
				pagination="true"
				@selection-change='gnbBatchSelect'>
				<template slot="toolbar">
                	<div class='toolbarHeadBtnBoxCls'>
                      	<!-- 已选数据 -->
                 		<div class="selectBlukBoxCls">
                             <div class="selectMain">
                             	 <span class="subTitle"><%=rb.getString("gNBSheBei")%></span>
	                             <div class="bulkSelectBtnBoxCls"  @click="gnbOpenBulkSelectTable">
									<span class="el-icon-selected el-icon"></span>
									<span class="bulkSelectNumBoxCls">( {{gnbDeviceSnSelection.length}} )</span>
								</div>
								<div class="selectTableBoxCls" v-show="gnbBulkSelectShow" style="position: absolute;top: 32px;left: 30px;">
                                     <div class="selectBoxTitle">
                                         <span><%=rb.getString("YiXuan")%></span>
                                         <span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="gnbCloseBulkSelectTable"></span>
                                     </div>
                                     <div class="selectBoxMain">
                                         <div class="tableInfoCls">
                                             <div class="tableInfoHeader">
                                                 <div><%=rb.getString("XiaoZhanBianMa")%></div>
                                                 <div @click="gnbClearBulkSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span><%=rb.getString("QingChu")%></div>
                                             </div>
                                             <el-ctable
                                                 id="bulkSelectTable"
                                                 ref="bulkSelectTable"
                                                 :data="gnbDeviceSnSelection"
                                                 :showHeader="false"
                                                 :rownumber="false"
                                                 :front-pagination="true"
                                                 height="270px" pagination="true" >
                                                 <el-table-column prop="id" v-if="false"></el-table-column>
                                                 <el-table-column width="588">
                                                     <template slot-scope="scope" >
                                                         <div class="tableItemCls">
                                                             <span>{{scope.row.serial_number}}</span>
                                                             <span @click="gnbDelBulkSelected(scope.row)" class="el-icon el-icon-circle-close item_show"></span>
                                                         </div>
                                                     </template>
                                                 </el-table-column>
                                             </el-ctable>
                                         </div>
                                     </div>
                                 </div>
                             </div>
                        </div>

                      	<div v-show="optBtnShow" :class="gnbDeviceSnSelection.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="gnbImportFileBtn">
                     		<span class='el-icon el-icon-operation-import'></span>
                     		<span><%=rb.getString("DaoRuWenJian")%></span>
                     	</div>
                     	<div :class="gnbDeviceSnSelection.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="gnbDownloadFileBtn">
                     		<span class='el-icon el-icon-operation-export'></span>
                     		<span><%=rb.getString("BackupRestoreDaoChuWenJian")%></span>
                     	</div>
                    </div>
                    <div class='commonFlex'>
                        <el-input placeholder="<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%>" suffic-icon='el-icon-search' v-model="gnbSearchText" style="width: 300px; margin: 10px 0 0 20px;">
                            <i slot="suffix" class="el-icon el-icon-common-search"  @click="gnbDeviceQuery" style='font-size: 16px; margin-top: 5px;'></i>
                        </el-input>

                        <div class="tableHeadQueryBoxCls">
                            <div v-for="(item,index) in advancedQueryItemList">
                                <div v-if="item.type == 'checkbox' && item.isShow" style="margin-right:10px;">
									<el-popfilter
										:label='item.label'
										v-model="item.checkedItemList"
										:list="item.options"
										:visible.sync="item.isShow"
										@check-change="advanceQuery(item.type,item.value,item.checkedItemList,'list')">
									</el-popfilter>
								</div>
								<div v-if="item.type == 'select' && item.isShow" style="margin-right:10px;">
									<el-popfilter
										type="single"
										:label='item.label'
										v-model="item.selectVal"
										:list="item.options"
										:visible.sync="item.isShow"
										@check-change="advanceQuery(item.type,item.value,item.selectVal,'list')">
									</el-popfilter>
								</div>
                            </div>
                            <div class="advancedQueryItemBox"  style="background: #FFF;" @click="clearFilterClick">
                                <%=rb.getString("QingKongShaiXuan")%>
                            </div>
                        </div>
                    </div>
                </template>
				<el-table-column label='' width="50" type="selection" :reserve-selection="true" prop="ck"></el-table-column>
				<el-table-column prop="connection_status" width="50">
					<template slot-scope="scope">
						<div :class="{
							'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
							'':scope.row.have_connected==2,
							'conn_exc':scope.row.connection_status=='Exception',
							'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
					</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serial_number" sortable></el-table-column>
				<el-table-column label='<%=rb.getString("HostName")%>' prop="host_name" sortable></el-table-column>
				<el-table-column label='<%=rb.getString("ChanPinLeiXing")%>' prop="product_type" sortable></el-table-column>
				<el-table-column label='<%=rb.getString("ZuiXinGengXinWenJian")%>' prop="latest_update_file"></el-table-column>
				<el-table-column label='<%=rb.getString("ZuiXianGengXinShiJian")%>' prop="latest_update_time" sortable></el-table-column>
			</el-ctable>
		</div>

		<!-- 下半部分 -->
		<div class="taskListWarp commonWarp" style="position:relative;margin-top: 10px;">
			<!--task list数据统计-->
			<div class="dataStatistics" v-if="gnbActiveName == 'taskList'">
				<div class="enableStatistics">
					<span class="enableTitleStatistics titleCommon"><i class="el-icon el-icon-status-waiting1 statusTip"></i><%=rb.getString("DengDai") %></span>
					<span class="enableNumStatistics titleCommon">{{gnbWaitingNum}}</span>
				</div>
				<div class="enableStatistics grantSuspended">
					<span class="enableTitleStatistics titleCommon"><i class="el-icon el-icon-status-inProgress statusTip"></i><%=rb.getString("JinXingZhong") %></span>
					<span class="enableNumStatistics titleCommon">{{gnbInProgressNum}}</span>
				</div>
				<div class="enableStatistics">
					<span class="enableTitleStatistics titleCommon"><i class="el-icon el-icon-status-suspend statusTip" ></i><%=rb.getString("ZanTing") %></span>
					<span class="enableNumStatistics titleCommon">{{gnbSuspendNum}}</span>
				</div>
				<div class="enableStatistics" style="margin-left:10px; border-right: 0;">
					<span class="enableTitleStatistics titleCommon"><i class="el-icon el-icon-status-terminate statusTip"></i><%=rb.getString("YiJieShu") %></span>
					<span class="enableNumStatistics titleCommon">{{gnbEndNum}}</span>
				</div>
			</div>
			<!-- device list 数据统计-->
			<div class="dataStatistics" v-if="gnbActiveName == 'deviceList'">
				<div class="enableStatistics">
					<div class="enableTitleStatistics titleCommon" >
						<i class="el-icon el-icon-circle-success deviceStatusTip"></i>
						<span style="color:#67D972"><%=rb.getString("ChengGong") %></span>
					</div>
					<span class="enableNumStatistics titleCommon">{{gnbSuccessNum}}</span>
				</div>
				<div class="enableStatistics" style="margin: 0 40px 0 10px;border-right: 0">
					<div class="enableTitleStatistics titleCommon">
						<i class="el-icon el-icon-circle-close deviceStatusTip"></i>
						<span class="redCol"><%=rb.getString("ShiBai") %></span>
					</div>
					<span class="enableNumStatistics titleCommon">{{gnbFailNum}}</span>
				</div>

				<div class="circleIcon placeholder-bt" style="top: -6px; right: 0;" placeholder="<%=rb.getString("DaoChu")%>">
					<span class="el-icon el-icon-circle-export" @click="gnbDeviceListExport"></span>
				</div>
			</div>
			<!--task list,device list tabs -->
			<el-tabs v-model="gnbActiveName" @tab-click='gnbTabClick' style="height:100%;" class='newTabs'>
				<el-tab-pane label="<%=rb.getString("RenWuLieBiao")%>" name="taskList">
					<!--task list table -->
			        <el-ctable ref="gnbTaskListTable" id="gnbTaskListTable" class='commonQueryToolbar'
			        	:time="6"
			        	height="100%"
			        	:url="gnbTaskListTableUrl"
			        	:query-params="gnbQueryTaskParams"
			        	@load-success="gnbTaskListTableLoadSuccess">
			          	<!-- 列表toolbar -->
			          	<template slot="toolbar">
							<div class='toolbarHeadBtnBoxCls commonQuery'>
								<el-query type="normal" @query="gnbTaskListQuery" placeholder="<%=rb.getString("RenWuMingCheng")%>"></el-query>
								<el-date-picker style='margin-left: 10px;'
									v-model="gnbDateValue"
									type="datetimerange"
									value-format="yyyy-MM-dd HH:mm:ss"
									range-separator="——"
									@change="gnbDateChange"
									start-placeholder='<%=rb.getString("KaiShiShiJian")%>'
									end-placeholder='<%=rb.getString("JieShuShiJian")%>'>
								</el-date-picker>

								<div class="tableHeadQueryBoxCls">
									<div v-for="(item,index) in gnbAdvancedQueryItemList">
										<div v-if="item.type == 'checkbox' && item.isShow" style="margin-right:10px;">
											<el-popfilter
												:label='item.label'
												v-model="item.checkedItemList"
												:list="item.options"
												:visible.sync="item.isShow"
												@check-change="advanceQuery(item.type,item.value,item.checkedItemList,'task')">
											</el-popfilter>
										</div>
										<div v-if="item.type == 'select' && item.isShow" style="margin-right:10px;">
											<el-popfilter
												type="single"
												:label='item.label'
												v-model="item.selectVal"
												:list="item.options"
												:visible.sync="item.isShow"
												@check-change="advanceQuery(item.type,item.value,item.selectVal,'task')">
											</el-popfilter>
										</div>
									</div>
									<div class="advancedQueryItemBox"  style="background: #FFF;" @click="gnbClearFilterClick">
										<%=rb.getString("QingKongShaiXuan")%>
									</div>
								</div>
							</div>
						</template>

			          	<!-- 列表columns -->
			          	<el-table-column prop="op" label=" " width="50" align="center" v-if="optBtnShow">
			            	<template slot-scope="scope">
			              		<div class="el-icon el-icon-operation-more" @click="gnbOptClick(scope.row,event)" v-clickoutside="gnbHanderClose"></div>
			            	</template>
			          	</el-table-column>
						<el-table-column label='<%=rb.getString("RenWuMingCheng")%>' min-width="260"  prop="TASK_NAME"></el-table-column>
			          	<el-table-column label='<%=rb.getString("CaoZuoRen")%>' min-width="100" prop="CREATE_USER"></el-table-column>
			          	<el-table-column label='<%=rb.getString("CaoZuoShiJian")%>' min-width="150" prop="CREATE_TIME"></el-table-column>
			          	<el-table-column label='<%=rb.getString("Type")%>' min-width="150" prop="TASK_TYPE"></el-table-column>
			          	<el-table-column label='<%=rb.getString("ZhuangTai")%>' min-width="150" prop="TASK_STATUS">
			          		<template slot-scope="scope">
			              		<div v-html="backupRestoreTaskTableStatus(scope.row.TASK_STATUS)"></div>
			            	</template>
			          	</el-table-column>
			          	<el-table-column label='<%=rb.getString("RenWuJinDu")%>' min-width="130" prop="TASK_PROGRESS"></el-table-column>
			          	<el-table-column label='<%=rb.getString("JieGuo")%>' min-width="150" prop="TASK_RESULT">
			            	<template slot-scope="scope">
			              		<div v-html="taskTableResult(scope.row.TASK_RESULT)"></div>
			            	</template>
			          	</el-table-column>
			          	<el-table-column label='<%=rb.getString("KaiShiShiJian")%>' min-width="150" prop="START_TIME"></el-table-column>
			          	<el-table-column label='<%=rb.getString("JieShuShiJian")%>' min-width="150" prop="END_TIME"></el-table-column>
			          </el-ctable>
			        <el-cmenu ref="gnbMenus" :data="gnbMenus" @click="gnbClickMenu"></el-cmenu>
				</el-tab-pane>

				<!--device list table  -->
				<el-tab-pane label="<%=rb.getString("BackupRestoreSheBeiLieBiao") %>" name="deviceList">
					<el-ctable ref="gnbDeviceListTable" id="gnbDeviceListTable" class='commonQueryToolbar'
						:url="gnbDeviceListTableUrl"
						:query-params="gnbResultParams"
						:time="6"
						height="100%"
						@load-success="gnbDeviceListTableLoadSuccess">
						<!-- 列表toolbar -->
						<template slot="toolbar">
							<div class='toolbarHeadBtnBoxCls commonQuery' style="height: 46px;">
								<el-query type="normal" @query="gnbDeviceListQuery" placeholder='<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%>/<%=rb.getString("RenWuMingCheng")%>'></el-query>
							</div>
			          	</template>
			      		<el-table-column prop="SERIAL_NUMBER" label="<%=rb.getString("XiaoZhanBianMa")%>" min-width="120"></el-table-column>
			      		<el-table-column prop="HOST_NAME" label="<%=rb.getString("HostName")%>" min-width="100"></el-table-column>
			      		<el-table-column prop="TASK_TYPE" label="<%=rb.getString("Type")%>" min-width="100"></el-table-column>
			      		<el-table-column prop="TASK_NAME" label="<%=rb.getString("RenWuMingCheng")%>" min-width="200"></el-table-column>
			      		<el-table-column prop="CONFIG_FILE" label="<%=rb.getString("BackupRestorePeiZhiWenJian")%>" min-width="200"></el-table-column>
			      		<el-table-column prop="PROGRESS_STATUS" label="<%=rb.getString("ZhuangTai")%>" min-width="80">
				          	<template slot-scope="scope">
				            	<div v-html="resultTableStatus(scope.row.PROGRESS_STATUS)"></div>
				          	</template>
			        	</el-table-column>
			      		<el-table-column prop="PROGRESS_RESULT" label="<%=rb.getString("JieGuo")%>" min-width="80">
			      			<template slot-scope="scope">
			              		<div v-html="resultTableResult(scope.row.PROGRESS_RESULT)"></div>
			            	</template>
			      		</el-table-column>
			      		<el-table-column prop="FAILURE_REASON" label="<%=rb.getString("ShiBaiYuanYin")%>" min-width="150"></el-table-column>
			      		<el-table-column prop="START_TIME" label="<%=rb.getString("KaiShiShiJian")%>" min-width="100"></el-table-column>
						<el-table-column prop="END_TIME" label="<%=rb.getString("JieShuShiJian")%>" min-width="100"></el-table-column>
			       </el-ctable>
				</el-tab-pane>
			</el-tabs>
		</div>
	</div>
	<el-slide class='commonBorderSlide' ref="gnbSlide"
		:url="gnbSlideUrl"
		:modal='gnbSlideModal'
		:title="gnbSlideTitle"
		:footer="gnbSlideFooter"
		:header='gnbSlideHeader'
		:position="gnbSlidePosition"
	 	:height="gnbSlideHeight"
	 	:width="gnbSlideWidth"
        :subloading="slideSubmitLoading"
	 	:ok-text="'<%=rb.getString("QueDing")%>'"
	 	:cancel-text="'<%=rb.getString("QuXiao")%>'"
	 	@cancel='gnbCancelSlide'
	 	@ok='gnbSaveSlide'>
	 </el-slide>

	<!-- 导入文件框 -->
	<el-dialog title='<%=rb.getString("PiLiangDaoRu")%>' :visible.sync="gnbShowConfirmInfo" width="1100" class="backupRestoreImport"
		:close-on-click-modal="false" @close="gnbCloseImport">
		<el-form :model="gnbImportForm" ref="gnbImportForm" label-position="left" :rules='gnbImportRules'>
			<el-form-item label="<%=rb.getString("DaoRuFangShi")%>" label-width="130px">
				<el-radio-group v-model="gnbImportModeSelection">
					<el-radio label="moreFile" class="moreFileSelected"><%=rb.getString("AnDaoRuWenJianMingPiPeiSheBei")%></el-radio>
					<el-radio label="singleFile" class="singleFileSelected">
					    <span><%=rb.getString("SuoXuanSheBeiPiPeiYiGePeiZhiWenJian")%></span>
					    <span class="oneFileMoreDevicesTip"><%=rb.getString("ZhiNengXuanZeXiangTongChanPinLeiXingDeSheBei")%></span>
					</el-radio>
				</el-radio-group>
			</el-form-item>
            <el-form-item label="<%=rb.getString("DaoRuWenJian")%>" label-width="130px" prop="gnbFileName" v-if="gnbImportModeSelection == 'moreFile'" class='importSelect'>
	            <el-upload
	           		ref="gnbMoreUpload"
	            	:multiple="true"
	            	:on-success='gnbMoreCheckFile'
	            	:on-change="gnbMoreFileChange"
	            	:show-file-list=false
				    :action="gnbImportForm.gnbUploadFileUrl"
				    :data="gnbMoreFileParams"
				    name="uploadFile"
				    :file-list="gnbMoreFileList"
				    :http-request="gnbMoreFileRequest"
				    :auto-upload="false"
				    :limit="20"
				    :on-exceed="gnbHandleExceed"
				    accept=".xml">
					<el-input :readonly="true" :value=gnbFileName placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>' class="w270" v-if="false">
						<a slot="suffix" class="el-icon el-icon-operation-import importBox" @click="gnbMoreFileSelect"></a>
					</el-input>
					<el-button size="small" type="primary" @click="gnbMoreFileSelect"><%=rb.getString("DianJiShangChuan")%></el-button>
					<span class="fileImportTypeTip"><%=rb.getString("DangQianZhiChixmlGeShi")%></span>
					<a slot="trigger" ref="file_up"></a>
				</el-upload>
            </el-form-item>

			<el-form-item label="<%=rb.getString("DaoRuWenJian")%>" label-width="130px" prop="gnbFileName" v-if="gnbImportModeSelection == 'singleFile'" class='importSelect'>
	            <el-upload
	            	ref="gnbUpload"
	            	:before-upload='gnbBeforeUpload'
	            	:on-success='gnbCheckFile'
	            	:on-change="gnbFileChange"
	            	:show-file-list=false
				    :action="gnbImportForm.gnbUploadFileUrl"
				    :data="gnbFileParams"
				    name="gnbUploadFile"
				    :auto-upload="false"
				    accept=".xml">
					<el-input :readonly="true" :value=gnbFileName placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>' class="w270">
						<a slot="suffix" class="el-icon el-icon-operation-import importBox" @click="gnbFileSelect"></a>
					</el-input>
					<span class="fileImportTypeTip"><%=rb.getString("DangQianZhiChixmlGeShiAll")%></span>
					<a slot="trigger" ref="file_up"></a>
				</el-upload>
            </el-form-item>

			<div style="padding-top:10px;border:1px solid #E9E9E9;" class="selectDeviceTable">
				 <div class='queryGroup'>
					<el-input v-model="gnbSearchValue" class='pairgrid-query' placeholder='<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%>'></el-input>
					<i @click="gnbDoFilter" class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
				</div>
				<el-table ref="newTable" :data="gnbTableData.slice((currentPage-1)*pageSize,currentPage*pageSize)" border @sort-change="gnbSortChange"
					style="height:246px;overflow: auto;flex: auto;margin-top:10px;">
					<el-table-column label="" type="index" min-width="50"></el-table-column>
					<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serial_number" sortable></el-table-column>
					<el-table-column label='<%=rb.getString("HostName")%>' prop="host_name" sortable></el-table-column>
					<el-table-column label='<%=rb.getString("ChanPinLeiXing")%>' prop="product_type"></el-table-column>
					<el-table-column label='<%=rb.getString("BackupRestorePeiZhiWenJian")%>' prop="file_name" v-if="gnbImportModeSelection == 'singleFile'"></el-table-column>
					<el-table-column label='<%=rb.getString("BackupRestorePeiZhiWenJian")%>' prop="file_name" v-if="gnbImportModeSelection == 'moreFile'" :formatter="gnbFileNameFormatter"></el-table-column>
					<el-table-column width="1" :show-overflow-tooltip="false">
						<template slot-scope="scope">
							<div slot="reference" class="opts-wrapper" style="overflow: hidden;">
								<i @click="gnbDeleteChecked(scope.row,scope.$index)" class="el-icon el-icon-circle-close"> </i>
							</div>
		          		</template>
					</el-table-column>
				</el-table>
				<el-pagination
					@size-change="gnbHandleSizeChange"
					@current-change="gnbHandleCurrentChange"
					:current-page="currentPage"
					:page-sizes="[50,100,200]"
					:pageSize="pageSize"
					layout="total,sizes, prev, pager, next, jumper"
					:total="gnbTableData.length">
				</el-pagination>
			</div>

			<div v-show="gnbNoSelectDevice" style="color:#FA5555;font-size:12px;"><%=rb.getString("ZhiShaoXuanZeYiGeSheBei")%></div>
			<div v-if="gnbImportModeSelection == 'singleFile'">
				<div v-show="gnbNoSameProductType" style="color:#FA5555;font-size:12px;"><%=rb.getString("ZhiNengXuanZeXiangTongChanPinLeiXingDeSheBei")%></div>
			</div>
		</el-form>

		<div slot="footer" class="dialog-footer">
			<div class="buttonGroup">
				<el-button type="primary" @click="gnbImportSubmit"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="gnbCloseImport"><%=rb.getString("QuXiao")%></el-button>
			</div>
		</div>
	</el-dialog>

</div>
</div>
<script>
	var gnbBackupRestoreVue = new Vue({
		el:'#gnbBackupRstore',
		data(){
			var vm = this,
            gnbValidateFilesName = function(rule,value,callback) {
            	var value = vm.gnbFileName;

				if(vm.gnbImportModeSelection == 'singleFile'){
                    if( value === '' || value === null || value === undefined) {
                        callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
                    }else if(!fileFormatMatch(value,"xml")){
                        callback(new Error('<%=rb.getString("DangQianZhiChixmlGeShiAll")%>'));
                    }else {
                        callback();
                    }
                }else{
                    if( value === '' || value === null || value === undefined) {
                        callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
                    }else {
                        callback();
                    }
                }
			};
			return{
				gnbPeriodDataLoading: true,  // 周期备份数据加载状态
				gnbWaitingNum:'0',
				gnbInProgressNum:'0',
				gnbSuspendNum:'0',
				gnbEndNum:'0',
				gnbSuccessNum:'0',
				gnbFailNum:'0',
				gnbDeviceSnSelection:[],
				gnbNewDeviceSnSelection:[],
				gnbTableData:[],
				gnbSearchValue:"",
				pageSize:50,
				currentPage:1,
				gnbShowConfirmInfo:false,
				gnbImportModeSelection:'moreFile',
				gnbPeriodSwitch:'0',
				gnbActiveName:'taskList',
				height:'100%',
				gnbDeviceUrl:'${ctx}/task/enb/config/backupRestore/queryCellInfos.action',
				//gnb devices 参数
				gNBDeviceParams:{
					timeZone: timeZone,
					product_type:'',
					searchText:'',
					likeFields: 'serial_number,host_name',
					isGnb: 1
				},
				product_type:'',
				gnbSearchText:'',

				gnbRowData:[],
				gnbTaskListTableUrl: '${ctx}/task/enb/config/backupRestore/queryTaskList.action',
				gnbDeviceListTableUrl: '${ctx}/task/enb/config/backupRestore/queryTaskDeviceList.action',

		       	gnbQueryTaskParams: {
		         	timeZone: timeZone,
		         	searchText: '',
		         	likeFields: 'task_name',
					taskName: '',
					createUser: '',
					taskType: '',
					taskStatus: '',
		         	startTime:'',
		         	endTime:'',
                    isGnb: 1
		       	},
		       	//device list  搜索
		       	gnbResultParams: {
		       		timeZone: timeZone,
		       		searchText: '',	
		       		likeFields: 'serial_number,task_name,host_name',
		       		isGnb: 1
		       	},
		       	gnbMenus: [],

		        gnbFileParams:{},
	            gnbFileName:'',
				gnbFileList:[],
				gnbImportForm: {
	                gnbUploadFileUrl: ''
	           	},

	           	//多设备多文件
	            gnbMoreFileParams:{},
	            gnbMoreFileList:[],
	            gnbFileData:[],
	            gnbOnePeriodData:{},
	            gnbImportRules: {
                    gnbFileName:[
                    	{validator: gnbValidateFilesName},
                    ],
	            },
	            gnbNoSelectDevice:false,
	            gnbNoSameProductType:false,

	            gnbSlideUrl:'',
				gnbSlideTitle:'',
				gnbSlideFooter:'',
				gnbSlideHeader:'',
				gnbSlidePosition:'',
				gnbSlideHeight:'',
				gnbSlideWidth:'',
                slideSubmitLoading:'',
				gnbSlideModal:'',
				gnbFirstOpenImport:false,
				gnbLastOpenImport:false,
				gnbCurSelectPageFlag:'',

				gnbBulkSelectShow: false,
				gnbDateValue:[],

				gnbAdvancedQueryItemList:[

					{
						type:'select',
						isShow:true,
						popoverShow:false,
						selectVal:'',
						label:'<%=rb.getString("Type") %>',
						options:[
							{value:'',label:'<%=rb.getString("QuanBu")%>'},
							{value:'1',label:'<%=rb.getString("BeiFen")%>'},
							{value:'2',label:'<%=rb.getString("HuiFu")%>'},
							{value:'3',label:'<%=rb.getString("ZhouQiBeiFen")%>'},
							{value:'4',label:'<%=rb.getString("HuiFuChuChangPeiZhi")%>'}
						],
						value:'taskType',
					},
					{
						type:'select',
						isShow:true,
						popoverShow:false,
						selectVal:'',
						label:'<%=rb.getString("ZhuangTai") %>',
						options:[
							{value:'',label:'<%=rb.getString("QuanBu")%>'},
							{value:'1',label:'<%=rb.getString("DengDai")%>'},
							{value:'2',label:'<%=rb.getString("JinXingZhong")%>'},
							{value:'3',label:'<%=rb.getString("ZanTing")%>'},
							{value:'4',label:'<%=rb.getString("YiJieShu")%>'}
						],
						value:'taskStatus',
					}
				],

				advancedQueryItemList:[
                    {
                        type:'checkbox',
                        isShow:true,
                        isIndeterminate:false,
                        checkAll:false,
                        popoverShow:false,
                        checkedItemList:[],
                        oldCheckedItemList:[],
                        label:'<%=rb.getString("ChanPinLeiXing") %>',
                        options:[],
                        value:'product_type',
                    },
                ],
			}
		},
		computed: {
			gnbIsSuperAdmin() {
				return is_super_user == 'true';
			},
			optBtnShow() {
				return writableMap['CODE_GNB_BACKUP_RESTORE'] == true;
			},
		},
		methods:{
		    backupOrRestoreInit(){
                var vm = this,
                    params={
                        isUpdatePW:"true",
                        isGnb: 1
                    };

                axios.post( '${ctx}/cell/version/getProductType.action', stringify(params)).then(function(response){
                    var data = response.data;

                    var arr = [];
                    (data || []).map(function(item){
                        if (item){
                            arr.push({label:item.name,value:item.value})
                        }
                    })
                    vm.advancedQueryItemList.map((items)=>{
                        if('product_type' == items.value){
                            items.options = arr;
                        }
                    })
                });
		    },
			// 获取周期备份任务信息（控制再次打开周期备份任务按钮以及开关的状态）
			gnbGetSwitchData(){
				var vm = this;
				
				// 设置加载状态为 true，禁用按钮
				vm.gnbPeriodDataLoading = true;

				axios.post('${ctx}/task/enb/config/backupRestore/periodTask.action?isGnb=1').then(function(response){
					var data = response.data;
					
					if(Object.keys(data).length != 0){
						vm.gnbOnePeriodData = data;
						vm.gnbPeriodSwitch = data.is_enable == 1 ? '1' : '0';
						
					}else{
						vm.gnbOnePeriodData = {};
						vm.gnbPeriodSwitch = '0';
					}
					
					// 数据加载完成，启用按钮
					vm.gnbPeriodDataLoading = false;
				}).catch(function(error){
					// 即使失败也要启用按钮，避免永久禁用
					vm.gnbPeriodDataLoading = false;
				})
			},
			gnbHandleExceed(files,fileList){
				this.$message.info('<%=rb.getString("ZuiDuoXuanZeWenJian20")%>');
			},

			//导入文件
			gnbImportFileBtn(){
				var vm = this, gnbSelectDatas = vm.gnbDeviceSnSelection;
				if(gnbSelectDatas.length != 0){
					vm.gnbShowConfirmInfo = true;
					gnbSelectDatas.map((item,index) => {
						var gnbSelectRowData = {
	      					serial_number: item.serial_number,
	      					host_name: item.host_name,
	      					product_type: item.product_type,
	      					file_name:''
	      		    	}
						vm.gnbTableData.push(gnbSelectRowData);
						vm.gnbNewDeviceSnSelection = vm.gnbTableData;
					})
				}else{
					vm.$message.info('<%=rb.getString("ZhiShaoXuanZeYiGeSheBei")%>')
				}
			},

			/**
			* 文件上传成功函数
			* @param res{object}   返回信息
			* @param file{object}  文件信息
			*/
			gnbMoreCheckFile(res,file){    //发送请求，校验device文件内容
				var vm = this;
				if(res.success){
					if(res.suc_count>0){
						vm.$message({
							type: 'success',
							message: '<%=rb.getString("ChengGong")%>'
						});
					}else {
						vm.$message({
							type: 'warning',
							message: '<%=rb.getString("ShiBai")%>'
						});
					}
					vm.gnbShowConfirmInfo = false;
					vm.$refs.gNBDeviceTable.refresh();
					vm.gnbMoreCloseFileSelect();
				}else{
					vm.$message({
						type: 'error',
						message: res.msg
					});
				}
				//修改已选择文件状态
				var gnbfileLists = vm.$refs.gnbMoreUpload.gnbUploadFile;
				gnbfileLists.forEach(function(file){
					file.status = 'ready';
				})
			},

			/**
			* 选择文件后，校验格式，并赋值页面显示
			* @param file{object}   文件信息
			* @param fileList{Array}  文件列表
			*/
			gnbMoreFileChange(file,fileList){
				var vm = this;
				var gnbNewFileName = file.name.substr(0,file.name.lastIndexOf("_CFG.xml"));// 截取的 _CFG.	xml之前的文件名称

	    	    vm.gnbTableData.map((item,index) => {
					if(item.serial_number == gnbNewFileName){
						item.file_name = file.name;
						vm.gnbFileName += file.name+',';
						vm.gnbMoreFileParams.fileName = file.name;
						vm.gnbFirstOpenImport = true;
					}
	    	    })

				vm.gnbLastOpenImport = true;
			},
			// 设备组名称格式化
			gnbFileNameFormatter(row,column,cellValue,index){
				var vm = this;
				if(vm.gnbFirstOpenImport == false){
					if(vm.gnbLastOpenImport == true){
						return '<%=rb.getString("WenJianMingChengYuJiZhanBianMaBuPiPei")%>';
					}else{
						return ''
					}
				}else{
					if(cellValue == '' || cellValue == null ){
						return '<%=rb.getString("WenJianMingChengYuJiZhanBianMaBuPiPei")%>';

					}else{
						return cellValue
					}
				}
			},

			// 选择文件
			gnbMoreFileSelect(){
				var vm =this;
				this.gnbTableData.map((item,index) => {
					item.file_name = '';
	    	    })
	    	    vm.gnbFileName = '';
	    	    vm.gnbFirstOpenImport = false;
				vm.$refs.gnbMoreUpload.clearFiles();
				vm.$refs['file_up'].click();
			},

			// 移除导入文件
			gnbMoreCloseFileSelect(){
				var vm = this;
				vm.gnbFileName = '';
				vm.$refs.gnbMoreUpload.clearFiles();
			},

			//提交时匹配的文件及文件流
			gnbMoreFileRequest(file){
				var vm = this;
   				var gnbNewFileName = file.file.name.substr(0,file.file.name.lastIndexOf("_CFG.xml"));// 截取的 _CFG.xml之前的文件名称
   				vm.gnbTableData.map((item,index) => {
					if(item.serial_number == gnbNewFileName){
						vm.gnbFileData.push(file.file)
					}
	    	    })
			},


			// --------------------------------------多设备导入一个文件--------------------------

			/**
			* 文件上传成功函数
			* @param res{object}   返回信息
			* @param file{object}  文件信息
			*/
			gnbCheckFile(res,file){    //发送请求，校验device文件内容
				var vm = this;
				if(res.success){
					if(res.suc_count>0){
						vm.$message({
							type: 'success',
							message: '<%=rb.getString("ChengGong")%>'
						});
					}else {
						vm.$message({
							type: 'warning',
							message: '<%=rb.getString("ShiBai")%>'
						});
					}
					vm.gnbShowConfirmInfo = false;
					vm.$refs.gNBDeviceTable.refresh();
					vm.gnbCloseFileSelect();
				}else{
					vm.$message({
						type: 'error',
						message: res.msg
					});
				}
				//修改已选择文件状态
				var gnbFilesList = vm.$refs.gnbUpload.gnbUploadFile;
				gnbFilesList.forEach(function(file){
					file.status = 'ready';
				})
			},

			/**
			* 选择文件后，校验格式，并赋值页面显示
			* @param file{object}   文件信息
			* @param fileList{Array}  文件列表
			*/
			gnbFileChange(file,fileList){
				var vm = this;
				vm.gnbFileName = file.name;
				vm.gnbFileParams.fileName = file.name;
				vm.gnbTableData.map((item,index) => {
					item.file_name = item.serial_number + "_CFG.xml"; //文件名称为 sn+'_CFG.xml';
	    	    })

			},

			// 选择文件
			gnbFileSelect(){
				var vm =this;
				
				this.gnbTableData.map((item,index) => {
					item.file_name = '';
	    	    })
	    	    vm.gnbFileName = '';
				vm.$refs.gnbUpload.clearFiles();
				vm.$refs['file_up'].click();
			},

			// 移除导入文件
			gnbCloseFileSelect(){
				var vm = this;
				
				vm.gnbFileName = '';
				vm.$refs.gnbUpload.clearFiles();
			},

			/**
			* 文件上传之前
			* @param file{object}   文件信息
			*/
			gnbBeforeUpload(file){
				var vm = this,
					gnbFileName = file.name,
					fileSize = file.size,
					fd = new FormData(),
					config = {
						headers: { 'Content-Type': 'multipart/form-data' }
					};
				var data = vm.gnbTableData;
				var cellCodes = '';
				if(data.length != 0) {
					data.map((item,index) => {
						if(item.serial_number != null){
							cellCodes += item.serial_number + ","
						}
					})
				}

				fd.append('uploadFile',file); //文件流
				fd.append('uploadFileName',gnbFileName);//文件名
				fd.append('fileSize',fileSize);//文件名	大小
				fd.append('serial_numbers',cellCodes);//基站编码

				axios.post("${ctx}/task/enb/config/backupRestore/importFile/oneToAny.action",fd,config).then(function(res){
					if(res.data["success"]){
						vm.$message.success('<%=rb.getString("ChengGong")%>');
						vm.$refs.gNBDeviceTable.refresh();
						vm.gnbShowConfirmInfo = false;
						vm.gnbFileList = [];
						vm.gnbFileName = '';
						vm.$refs.gnbImportForm.resetFields();
					}else{
						vm.$message.error(res.data["message"]);
						vm.gnbShowConfirmInfo = false;
					}
				})

				return false;
			},


			/*确定导入*/
	        gnbImportSubmit() {
				var vm = this, noSelectData;

				if(vm.gnbTableData.length == 0 ){
					vm.gnbNoSelectDevice = true;
					noSelectData = false;
				}else{
					vm.gnbNoSelectDevice = false;
					noSelectData = true;
				}

				//多设备，单文件
				if(vm.gnbImportModeSelection == 'singleFile'){
					var productType = vm.gnbTableData.map(function(item){
						return item.product_type;
					});
					var newProductList = productType.filter(function(item){
						return item !== '';
					});
					var lastArr = Array.from(new Set(newProductList));
					//只能选择相同产品类型的设备，如果已勾选数据无产品类型时，可忽略此数据，但是给后端传参（serial_numbers）包含这些无产品类型的数据
					if(lastArr.length > 1){
						vm.gnbNoSameProductType = true;
						noSelectData = false;
					}else{
						vm.gnbNoSameProductType = false;
						noSelectData = true;
					}

					vm.$refs.gnbImportForm.validate((valid) => {
	                    if (valid && noSelectData) {
	                    	vm.$refs.gnbUpload.submit();
	                    }
	                })
				}else{
					//多设备  多文件
					vm.$refs.gnbImportForm.validate((valid) => {
	                    if (valid && noSelectData) {
	    					var fd = new FormData(),
		    					config = {
		    						headers: { 'Content-Type': 'multipart/form-data' }
		    					};

		                    vm.$refs.gnbMoreUpload.submit();
		    				var data = vm.gnbTableData;
		    				var cellCodes = '';
		    				if(data.length != 0) {
		    					data.map((item,index) => {
		    						if(item.serial_number != null){
		    							cellCodes += item.serial_number + ","
		    						}
		    					})
		    				}

		    				var gnbFileNames = '';
		    				vm.gnbFileData.forEach(file=>{
		    					fd.append('uploadFile',file,file.name);
		    					gnbFileNames += file.name + ","
		    				})

		    				fd.append('uploadFileName',gnbFileNames);//文件名
		    				fd.append('serial_numbers',cellCodes);//基站编码
		    				axios.post("${ctx}/task/enb/config/backupRestore/importFile/anyToAny.action",fd,config).then(function(res){
		    					if(res.data["success"]){
		    						vm.$message.success('<%=rb.getString("ChengGong")%>');
		    						vm.$refs.gNBDeviceTable.refresh();
		    					}else{
		    						vm.$message.error(res.data["message"]);
		    					}
		    					vm.gnbCloseImport();
		    				})
                    	}
                	})

				}

			},
			// 关闭导入弹出框
			gnbCloseImport(){
				var vm = this;
				
				vm.gnbShowConfirmInfo = false;
				vm.gnbFileList = [];
				vm.gnbFileName = '';
				vm.$refs.gnbImportForm.resetFields();
				vm.gnbTableData =[];
				vm.gnbNoSelectDevice = false;
				vm.gnbNoSameProductType = false;
				vm.gnbImportModeSelection = 'moreFile';
				vm.gnbFileData =[];
				vm.$refs['gNBDeviceTable'].clearSelection();
				vm.gnbSearchValue = '';
				vm.gnbNewDeviceSnSelection = [];
				vm.gnbFirstOpenImport = false;
				vm.gnbLastOpenImport = false;
			},

			//批量导入-表格搜索
			gnbDoFilter(){
				var vm = this, gnbSearchInput = vm.gnbSearchValue, serchArray = [];

				if(gnbSearchInput === '' || gnbSearchInput === null || gnbSearchInput === undefined){
					vm.gnbUpdateImportTable(vm.gnbNewDeviceSnSelection);
				}else{
					vm.gnbTableData.forEach(function(item){
						if(item.serial_number != null){
							if(item.serial_number.indexOf(gnbSearchInput) > -1 || item.host_name.indexOf(gnbSearchInput) > -1){
								serchArray.push(item)
							}
						}
					})
					vm.gnbUpdateImportTable(serchArray);
				}
			},
			gnbUpdateImportTable(arr){
				var vm = this;

				vm.gnbTableData = arr;
				vm.gnbTableData.map((item,index) => {
					item.file_name = item.file_name;
	    	    })
				vm.currentPage = 1;
			},

			//批量导入-删除
			gnbDeleteChecked(row,index){
				var vm = this,
					index = index,
					fileAllName=[];

				if(index >= 0) {
					vm.gnbTableData.splice(index,1);
			    	vm.gnbNewDeviceSnSelection.map((item,index) => {
						if(item.serial_number == row.serial_number ){
							vm.gnbNewDeviceSnSelection.splice(index,1);
						}
			    		//多设备多文件
			    		if(vm.gnbImportModeSelection == 'moreFile'){
			    			vm.gnbFileName = '';

							if(item.file_name == '' || item.file_name == null || item.file_name == 'undefined'){

							}else{
								fileAllName.push(item.file_name)
							}
				    		vm.gnbFileName = fileAllName.join(',');
						}
			    	})
			    }
			},

			gnbHandleSizeChange(val){
				this.pageSize = val;
			},
			gnbHandleCurrentChange(val){
				this.currentPage = val;
			},
			currentChangePage(list){
				var fromNum = (this.currentPage - 1) * this.pageSize;
				var toNum = this.currentPage * this.pageSize;
				//this.tableDataEnd = [];
				for(;fromNum < toNum; fromNum++){
					if(list[fromNum]){
						this.gnbTableData.push(list[fromNum])
					}
				}
			},

			//gNB devices 批量下载
			gnbDownloadFileBtn(){
				var vm = this, gnbFileNames = '';

				if(vm.gnbDeviceSnSelection.length != 0){
					vm.gnbDeviceSnSelection.map(function(item){
						if(item.latest_update_file !== '' && item.latest_update_file !== null && item.latest_update_file !== undefined){
							gnbFileNames += item.latest_update_file + ",";
						}
					})

					if(gnbFileNames == '' || gnbFileNames == null || gnbFileNames == undefined){
						vm.$message.info('<%=rb.getString("DangQianWuPiPeiWenJian")%>')
					}else{
						exportByForm("${ctx}/task/enb/config/backupRestore/batch/exportFile.action",{
							fileNames: gnbFileNames,
							isGnb: 1
				        });
					}
				}else{
					vm.$message.info('<%=rb.getString("ZhiShaoXuanZeYiGeSheBei")%>')
				}

			},
			//新建备份任务
			gnbBackupTaskBtn(){
				var vm = this;

				vm.gnbSlideHeader = true;
	    	    vm.gnbSlideTitle = '<%=rb.getString("XinJianRenWu")%>';
	    	    vm.gnbSlideUrl = '${ctx}/task/profile/backupRestore/goGNBAddBackupRestoreTaskPage.action?taskType=backup';
	    	    vm.gnbSlideFooter = 'true';
	    	    vm.gnbSlidePosition = 'top';
	    	    vm.gnbSlideHeight = '100%';
	    	    vm.gnbSlideWidth = '100%';
                vm.slideSubmitLoading = false;
	    	    vm.gnbCurSelectPageFlag = 'backup';
	    	    vm.$refs.gnbSlide.showSlide(function(){
					eventBus.$emit("show-gnbType",'backup','');
	    	    });
			},
			//新建恢复任务
			gnbRestoreBtn(){
				var vm = this;

				vm.gnbSlideHeader = true;
	    	    vm.gnbSlideTitle = '<%=rb.getString("XinJianRenWu")%>';
	    	    vm.gnbSlideUrl = '${ctx}/task/profile/backupRestore/goGNBAddBackupRestoreTaskPage.action?taskType=restore';
	    	    vm.gnbSlideFooter = 'true';
	    	    vm.gnbSlidePosition = 'top';
	    	    vm.gnbSlideHeight = '100%';
	    	    vm.gnbSlideWidth = '100%';
                vm.slideSubmitLoading = false;
	    	    vm.gnbCurSelectPageFlag = 'restore';
	    	    vm.$refs.gnbSlide.showSlide(function(){
					eventBus.$emit("show-gnbType",'restore','');
	    	    });
			},
			
			//新建周期备份任务
			gnbPeriodBackupBtn(){
				var vm = this;
				
				// 如果数据还在加载中，不允许点击
				if(vm.gnbPeriodDataLoading){
					return;
				}
				
				var gnbUpdateOrAdd = Object.keys(vm.gnbOnePeriodData).length;

				if(gnbUpdateOrAdd == 0){
					//新建周期备份任务
					vm.gnbSlideHeader = true;
		    	    vm.gnbSlideTitle = '<%=rb.getString("XinJianZhouQiBeiFen")%>';
		    	    vm.gnbSlideUrl = '${ctx}/task/profile/backupRestore/goGNBAddBackupRestoreTaskPage.action?taskType=backup';
		    	    vm.gnbSlideFooter = true;
		    	    vm.gnbSlidePosition = 'top';
		    	    vm.gnbSlideHeight = '100%';
		    	    vm.gnbSlideWidth = '100%';
                    vm.slideSubmitLoading = false;
		    	    vm.gnbCurSelectPageFlag = 'periodBackup';
		    	    vm.$refs.gnbSlide.showSlide(function(){
						eventBus.$emit("show-gnbType",'periodBackup','');
		    	    });
				}else{
					//修改周期备份任务
					vm.gnbSlideHeader = true;
		    	    vm.gnbSlideTitle = '<%=rb.getString("XiuGaiZhouQiBeiFen")%>';
		    	    vm.gnbSlideUrl = '${ctx}/task/profile/backupRestore/goGNBAddBackupRestoreTaskPage.action?taskType=backup';
		    	    vm.gnbSlideFooter = true;
		    	    vm.gnbSlidePosition = 'top';
		    	    vm.gnbSlideHeight = '100%';
		    	    vm.gnbSlideWidth = '100%';
                    vm.slideSubmitLoading = false;
		    	    vm.gnbCurSelectPageFlag = 'updatePeriodBackup';
		    	    vm.$refs.gnbSlide.showSlide(function(){
		    	    	eventBus.$emit("show-gnbType",'updatePeriodBackup',vm.gnbOnePeriodData);
		    	    });
				}
			},
		
			//device list 导出
			gnbDeviceListExport(){
				var vm =this;

				exportByForm("${ctx}/task/enb/config/backupRestore/exportTaskDeviceList.action",{
					timeZone: timeZone,
					likeFields: 'serial_number,task_name,host_name',
					searchText: vm.gnbResultParams.searchText,
					isGnb: 1
				});
			},
			// task list 数据统计
			gnbTaskListTableLoadSuccess(data){
				var vm = this;

				if(data.properties){
					vm.gnbWaitingNum = data.properties.waiting;
					vm.gnbInProgressNum = data.properties.inProgress;
					vm.gnbSuspendNum = data.properties.suspend;
					vm.gnbEndNum = data.properties.end;
				}
			},
			// device list 数据统计
			gnbDeviceListTableLoadSuccess(data){
				var vm = this;

				if(data.properties){
					vm.gnbSuccessNum = data.properties.success;
					vm.gnbFailNum = data.properties.fail;
				}
			},

		    gnbOptClick(row,ev){
	     		var vm = this, status = row.TASK_STATUS;

	     		//vm.taskStatus = row.TASK_STATUS;
	     		vm.$root.gnbRowData = row;
	     		vm.gnbMenus= [
			          {label:'<%=rb.getString("KaiShi")%>',cls:"CODE_GNB_BACKUP_RESTORE hidden",code:'start'},
			          {label:'<%=rb.getString("ZanTing")%>',cls:"CODE_GNB_BACKUP_RESTORE hidden" ,code:'stop'},
			          {label:'<%=rb.getString("ZhongZhiRenWu")%>',cls:"CODE_GNB_BACKUP_RESTORE hidden",code:'terminate'},
			          {label:'<%=rb.getString("ShanChu")%>',cls:"CODE_GNB_BACKUP_RESTORE hidden",code:'del'}
			    ]

		    	initTaskStatus(status,vm.gnbMenus);
	     		vm.$nextTick(function(){
		    		document.body.click();
    		    	vm.$refs.gnbMenus.show(ev);
		    	});
		    },

		    /**
			 * 点击操作出现下拉菜单
			 * @param ev: 点击当前项数据是哪个
			 * */
    	    gnbClickMenu(ev){
				var vm = this,
    	    	codes = {
    	    		start: vm.gnbStartRebootTask,
    	    		stop: vm.gnbSuspendRebootTask,
    	    		terminate: vm.gnbTerminateRebootTask,
    	    		del: vm.gnbDelRebootTask
    	    	}

    	    	if(codes[ev.code]){
    	    		codes[ev.code](vm.$root.gnbRowData["TASK_ID"], vm.$root.gnbRowData["TASK_STATUS"])
    	    	}
    	    },

    	    /**
			 * 开始执行任务
			 * @param task_id: 当前数据ID
			*/
			gnbStartRebootTask(task_id){
    	    	var vm = this;

    	    	axios.post('${ctx}/task/enb/config/backupRestore/activeTask.action',stringify({
    	    		taskId : task_id,
    	    	})).then(function(response){
    	    		var data = response.data;
    	    		if(data["success"]){
    	    			vm.$message.success('<%=rb.getString("ChengGong")%>');
    	    			//vm.$message.success(data["message"]);// 接口返回的message为 '';
    	    			vm.$refs.gnbTaskListTable.refresh()
    	    		}else{
    	    			vm.$message.error(data["message"])
    	    		}
    	    	})
    	    },
			/**
			 * 暂停任务
			 * @param task_id: 当前数据ID
			*/
    	    gnbSuspendRebootTask(task_id){
    	    	var vm = this;

    	    	axios.post('${ctx}/task/enb/config/backupRestore/suspendTask.action',stringify({
    	    		taskId : task_id,
    	    	})).then(function(response){
    	    		var data = response.data;
    	    		if(data["success"]){
    	    			//vm.$message.success('<%=rb.getString("ChengGong")%>');
    	    			vm.$message.success(data["message"]);
    	    			vm.$refs.gnbTaskListTable.refresh()
    	    		}else{
    	    			vm.$message.error(data["message"])
    	    		}
    	    	})
    	    },
			/**
			 * 终止任务函数
			 * @param task_id: 当前数据ID
			*/
    	    gnbTerminateRebootTask(task_id){
    	    	var vm = this;
    	    	axios.post('${ctx}/task/enb/config/backupRestore/terminateTask.action',stringify({
    	    		taskId : task_id,
    	    	})).then(function(response){
    	    		var data = response.data;
    	    		if(data["success"]){
    	    			//vm.$message.success('<%=rb.getString("ChengGong")%>');
    	    			vm.$message.success(data["message"]);
    	    			vm.$refs.gnbTaskListTable.refresh()
    	    		}else{
    	    			vm.$message.error(data["message"])
    	    		}
    	    	})
    	    },
			/**
			 *   删除任务
			 * @param task_id: 当前数据ID
			*/
    	    gnbDelRebootTask(task_id){
    	    	var vm = this;
    	    	this.$confirm('<%=rb.getString("QueRenShanChuRenWu")%>',QueRen,{
    	    		customClass:'warningConfirm',
    	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
    	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
    	    		type:'warning',
    	    		closeOnClickModal:false
    	    	}).then(() => {
    	    		axios.post('${ctx}/task/enb/config/backupRestore/delTask.action',stringify({
    		    		taskId:task_id,
    		    	})).then(function(response){
    		    		var data = response.data;
    		    		if(data["success"]){
    		    			vm.$refs.gnbTaskListTable.refresh()
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


			/**
			 * 多选响应
			 * @param selection:选择的数据
			*/
		    gnbBatchSelect(selection){
		    	var vm = this;

		    	vm.gnbDeviceSnSelection = selection;
			},
			gnbTabClick() {
				this.$nextTick(function(){
					document.body.click();
				})
			},

			// gNB devices 表格搜索
			gnbDeviceQuery(){
				var vm = this;

				this.gNBDeviceParams.searchText = this.gnbSearchText;
			},
			// gNB devices 搜索
			gnbDeviceListQuery(val) {
	       		var vm = this;

	       		vm.gnbResultParams.searchText = val;
	     	},
	     	// task list  搜索
			gnbTaskListQuery(val) {
	       		var vm = this;

				Object.assign(vm.gnbQueryTaskParams, {
					searchText: val,
					//startTime: '',
					//endTime: ''
				});
	     	},
			//gnbSlide 保存
			gnbSaveSlide(){
				eventBus.$emit('taskSave-gnbOk')
			},
			//gnbSlide 取消
			gnbCancelSlide(){
				eventBus.$emit('hander-gnbCancel')
			},

			gnbCloseBackupRestoreTask(){
				var vm = this;

		    	vm.$refs.gnbSlide.hide();
				vm.$refs.gNBDeviceTable.clearSelection();
                vm.gnbBulkSelectShow = false;
		    	vm.$refs.gnbTaskListTable.refresh();
		    	vm.$refs.gnbDeviceListTable.refresh();
		    	vm.$refs.gNBDeviceTable.refresh();
		    	vm.gnbGetSwitchData();
		    },
		 	//点击页面其他地方关闭菜单
	        gnbHanderClose(){
				var vm =this;

	            vm.$refs.gnbMenus.hide();
	        },
	        //排序点击
	        gnbSortChange(data){
				var vm =this;

				vm.currentPage = 1;
				if(data.prop =='serial_number'){
					vm.gnbTableData = vm.gnbTableData.sort(vm.gnbCompare(data.prop,data.order == 'ascending'))
				}else if(data.prop =='host_name'){
					vm.gnbTableData = vm.gnbTableData.sort(vm.gnbCompare(data.prop,data.order == 'ascending'))
				}
				vm.gnbTableData = vm.gnbTableData.slice(0,vm.pageSize)
			},
			gnbCompare(attr,rev){
				if(rev == undefined){
					rev = 1
				}else{
					rev = (rev) ? 1 : -1
				}
				return function(a,b){
					a = a[attr];
					b = b[attr];
					if(a < b){
						return rev * -1;
					}
					if(a> b){
						return rev * 1
					}
				}
			},

			gnbDateChange(val) {
				var vm = this;

				vm.gnbDateValue = val;
				if(val != null){
					vm.gnbQueryTaskParams.startTime = vm.gnbDateValue[0];
					vm.gnbQueryTaskParams.endTime = vm.gnbDateValue[1];
				}
			},
			//------------------------------已选
			// 打开已选弹窗
            gnbOpenBulkSelectTable(){
                var vm = this;

                vm.gnbBulkSelectShow = true
            },
            // 关闭已选弹窗
            gnbCloseBulkSelectTable(){
                var vm = this;

                vm.gnbBulkSelectShow = false;
            },
            // 设备已选表格 清空事件
            gnbClearBulkSelected(){
                var vm = this;

                vm.$refs["gNBDeviceTable"].clearSelection();
                vm.gnbBulkSelectShow = false;
            },
            // 设备已选表格 单个删除事件
            gnbDelBulkSelected(rows){
                var vm = this,
					tabs = 'gNBDeviceTable',
					rowKey = 'serial_number';

				vm.gnbDeviceSnSelection = vm.gnbDeviceSnSelection.filter((items)=>{
					return items[rowKey] != rows[rowKey]
				});
				var selection = this.$refs[tabs].$refs.ctableInner.store.states.selection,
					irow= selection.filter((items)=>{
						return items[rowKey] == rows[rowKey]
					})[0];
				vm.$refs[tabs].toggleRowSelection(irow,false);
				var idx = vm.$refs[tabs].ckList.indexOf(rows[rowKey]);
				vm.$refs[tabs].ckList.splice(idx,1);

                if(vm.gnbDeviceSnSelection.length == 0){
                	vm.gnbBulkSelectShow = false;
                }
            },
            //-------------------------------------------------下拉菜单查询
			// 高级查询 确定事件
			advanceQuery(type,paramsItem,value,optType){
				var vm = this,
					params ={};
				if(type == 'select'){
					params[paramsItem] = value;
				}else{
					params[paramsItem] = value.join(',');
				}
				if(optType == 'task'){
					Object.assign(vm.gnbQueryTaskParams, params);
				}else{
					Object.assign(vm.gNBDeviceParams, params);
				}
			},
    		// 清除筛选
    		gnbClearFilterClick(){
    			var vm = this, params = {};

    			vm.gnbAdvancedQueryItemList.map((items)=>{
    				if(items.isShow && items.isShow== true ){
    					items.selectVal = '';
    					params[items.value] = '';
    				}
    			})
    			Object.assign(vm.gnbQueryTaskParams, params);
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
                Object.assign(vm.gNBDeviceParams, params);
                document.body.click();
            },
		},
		watch:{
			gnbImportModeSelection(val){
				this.gnbFileData =[];
				this.gnbSearchValue = '';
				this.gnbFileName = '';
				this.gnbTableData.map((item,index) => {
					item.file_name = '';
	    	    })
	    	    this.gnbFirstOpenImport = false;
				this.gnbLastOpenImport = false;
	    	    this.gnbUpdateImportTable(this.gnbDeviceSnSelection);
			},
			gnbDateValue(newVal){
				var vm = this;
				if(!newVal){
					newVal = [];
					vm.gnbQueryTaskParams.startTime = '';
	    			vm.gnbQueryTaskParams.endTime = '';
				}
			},
		},
		mounted(){
			this.gnbGetSwitchData();
		    this.backupOrRestoreInit();
			eventBus.$off("cancel-gnbBackupRestore").$on('cancel-gnbBackupRestore', this.gnbCloseBackupRestoreTask);
		}
	})
</script>
