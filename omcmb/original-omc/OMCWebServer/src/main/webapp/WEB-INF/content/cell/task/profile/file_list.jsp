<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>

<style>
	#backupRestore .enbDeviceWarp,
	#backupRestore .taskListWarp {
		flex:1;
		overflow:auto;
	}
	
	#backupRestore .el-checkbox__label{
		font-size:12px;
	}

	#backupRestore .filter-opt{
		width:20px;
		height:20px;
	}
	#backupRestore .dataStatistics{
		position:absolute;
		top:52px;
		right:10px;
		height:16px;
		z-index:99;
		display:flex;
	}
	#backupRestore .dataStatistics .grantSuspended{
		margin:0 10px;
	}
	#backupRestore .dataStatistics .enableStatistics{
		display:flex;
		height:16px;
		line-height:16px;
		border-right: 1px solid #D5DCEC;
	}
	#backupRestore .dataStatistics .enableStatistics .enableTitleStatistics{
		color:#4D84FF;
	}
	#backupRestore .dataStatistics .enableStatistics .enableNumStatistics{
		color: rgba(0, 0, 0, 0.8);
	}
	#backupRestore .dataStatistics .enableStatistics .titleCommon{
		padding:0 10px 0 6px;
		font-size:14px;
	}

	#backupRestore .statusTip{
		margin-right:6px;
	}
	#backupRestore .deviceStatusTip{
		margin-right:8px;
	}
	#backupRestore .dataStatistics .statusTip:before,
	#backupRestore .dataStatistics .deviceStatusTip:before{
		font-size:16px;
	}
	#backupRestore .el-icon-circle-success:before{
		color:#67D972;
	}
	#backupRestore .dataStatistics .el-icon-circle-close:before{
		color:#E88282;
		content:'\e6fb';
	}

	#backupRestore .w270{
		width:400px;
	}

	#backupRestore .backupRestoreImport .el-form-item{
		margin-bottom:22px;
	}
	#backupRestore .backupRestoreImport .el-dialog__body{
		height:450px;
		padding:20px 30px 0;
	}
	#backupRestore .backupRestoreImport .el-dialog__body .importSelect .el-form-item__label{
		margin-top:-7px;
	}
	#backupRestore .backupRestoreImport .el-dialog__body .el-input__suffix{
		top:4px;
	}
	#backupbackupRestoreRstore .backupRestoreImport .el-dialog__body .el-form-item__error{
		padding-top:0;
	}
	#backupRestore .backupRestoreImport .el-dialog__footer{
		text-align:left;
		padding:30px 30px;
	}
	#backupRestore .backupRestoreImport .el-dialog__body thead tr th:first-child .cell{
		display:none;
	}
	#backupRestore .advanceQuery .el-input__inner{
		color:#333333;
	}
	#backupRestore .selectDeviceTable .queryGroup{
		margin-left:10px;
	}
	#backupRestore .queryGroup .el-icon-common-search{
		margin-top:-3px;
	}

	#backupRestore .queryGroup,
	#backupRestore .pairgrid-query .el-input__inner{
		height:24px;
	}
	
	#backupRestore .column-flex .el-checkbox{
		display: block;
		margin-left: 0px;
		padding-top:10px;
	}
	
	#backupRestore .warningConfirm .el-button--primary{
		background-color:#FFFFFF !important;
		color:#666666 !important;
		border-color:#DCDFE6 !important;
	}

	#backupRestore .el-tabs--card>.el-tabs__header .el-tabs__nav{
		margin-left:10px;
		border:none; 		
	}
	#backupRestore input::-webkit-input-placeholder{
		color:#CCCCCC !important;
	}
	#backupRestore input::-moz-input-placeholder{
		color:#CCCCCC !important;
	}
	#backupRestore input::-ms-input-placeholder{
		color:#CCCCCC !important;
	}
	#backupRestore .circleIcon{
		min-width:26px;
	}
	#backupRestore .el-message--info{
		border-color:#81A8FF;
		background-color:#F2F6FF;
	}
	#backupRestore .el-ctable thead tr th{
		color:#333333;
	}
	#backupRestore .el-ctable tbody tr td{
		color:#666666;
	}
	#backupRestore .el-table .descending .sort-caret.descending{
		border-top-color:#4D84FF;
	}
	#backupRestore .el-dialog__header .el-dialog__headerbtn{
		right:20px;
		top:13px;
	}
	#backupRestore .el-message__icon{
		margin-right:10px;
	}
	#backupRestore .importFileWarp{
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
	#backupRestore .importFileWarp span:first-child{
		font-size:16px;
		margin-top:4px;
		margin-right:5px;
	}
	#backupRestore .importFileWarp span:last-child{
		font-size:12px;
	}
	#backupRestore .moreFileSelected{
		padding-top:12px;
		display: block;
	}
	#backupRestore .singleFileSelected{
		padding-top:12px;
		display: block;
		margin-left: 0;
	}
	#backupRestore .oneFileMoreDevicesTip{
		text-align:left;
		margin-left:10px;
		color: rgba(0, 0, 0, 0.32);
	}

	#backupRestore .fileImportTypeTip{
		color: rgba(0, 0, 0, 0.32);
		margin-left:10px;
	}
	#backupRestore .el-radio__inner:hover{
		border-color:#4D84FF;
	}
	#backupRestore .el-date-editor .el-range__close-icon {
		line-height:20px;
	}
	#profileAddDiv .el-icon-not-installed:before {
		color: #A0B2C8;
	}
	#profileAddDiv .el-icon-status-timeOut:before { 
		color: #19D5F3;
	}
</style>
<div class="overflow-cls">
<!-- eNB备份与恢复 -->
<div id="backupRestore" style="min-width: 1260px; width: 100%; height: 100%;">
	<div class='panelDefault' style="background:unset; border: 0; display:flex; flex-direction:column;">
		<div class="enbDeviceWarp commonWarp">	
			<!-- 右上角操作按钮 -->	
			<div>				
				<div v-if="isSuperAdmin">
					<div @click="backupTaskBtn" class="circleIcon placeholder-bt CODE_ENB_BACKUP_RESTORE hidden" style="right: 100px;top:6px;" placeholder="<%=rb.getString("BeiFen")%>">
						<span class="el-icon el-icon-circle-backup"></span>
					</div>
					<div id='profileAddDiv' @click="periodbackupBtn" class="circleIcon placeholder-bt" :class="{'is-disabled': periodDataLoading}" style="right: 60px;top:6px;cursor:pointer" :style="{cursor: periodDataLoading ? 'not-allowed' : 'pointer', opacity: periodDataLoading ? 0.6 : 1}" placeholder="<%=rb.getString("ZhouQiBeiFens")%>">		
						<span class="el-icon el-icon-circle-backup"></span>
						<span v-if='periodSwitch == "0"' class='el-icon el-icon-not-installed' style='position: absolute; top: 7px; right: 0; font-size: 14px;'></span>
						<span v-if='periodSwitch == "1"' class='el-icon el-icon-status-timeOut' style='position: absolute; top: 7px; right: 0; font-size: 14px;'></span>
					</div>
					
					<div id='profileViewDiv' @click="restoreBtn" class="circleIcon placeholder-bt CODE_ENB_BACKUP_RESTORE hidden" style="top:6px;right:20px;" placeholder="<%=rb.getString("HuiFu")%>">		
						<span class="el-icon el-icon-circle-restore"></span>
					</div>
				</div>
				<div v-else>
					<div @click="backupTaskBtn" class="circleIcon placeholder-bt CODE_ENB_BACKUP_RESTORE hidden" style="right: 60px;top:6px;" placeholder="<%=rb.getString("BeiFen")%>">
						<span class="el-icon el-icon-circle-backup"></span>
					</div>
					<div id='profileViewDiv' @click="restoreBtn" class="circleIcon placeholder-bt CODE_ENB_BACKUP_RESTORE hidden" style="top:6px;right:20px;" placeholder="<%=rb.getString("HuiFu")%>">		
						<span class="el-icon el-icon-circle-restore"></span>
					</div>
				</div>
			</div>
			
			<el-ctable ref="eNBDeviceTable" 
				id="eNBDeviceTable" 
				row-key="small_cell_code"
				:limit="100"
				:time="6"
				:url="deviceUrl" 
				:height="height" 
				:query-params="eNBDeviceParams" 
				pagination="true" 
				@selection-change='batchSelect'>						
				<template slot="toolbar">
                	<div class='toolbarHeadBtnBoxCls' style='margin: -10px 0 0 0;'>
                      	<!-- 已选数据 -->
                 		<div class="selectBlukBoxCls">
                             <div class="selectMain">
                             	 <span class="subTitle"><%=rb.getString("ENBSheBei")%></span>
	                             <div class="bulkSelectBtnBoxCls"  @click="openBulkSelectTable">
									<span class="el-icon-selected el-icon"></span>
									<span class="bulkSelectNumBoxCls">( {{deviceSnSelection.length}} )</span>
								</div>
								<div class="selectTableBoxCls" v-show="bulkSelectShow" style="position: absolute;top: 32px;left: 30px;">
                                     <div class="selectBoxTitle">
                                         <span><%=rb.getString("YiXuan")%></span>
                                         <span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="closeBulkSelectTable"></span>
                                     </div>
                                     <div class="selectBoxMain">
                                         <div class="tableInfoCls">
                                             <div class="tableInfoHeader">
                                                 <div><%=rb.getString("XiaoZhanBianMa")%></div>
                                                 <div @click="clearBulkSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span><%=rb.getString("QingChu")%></div>
                                             </div>
                                             <el-ctable 
                                                 id="bulkSelectTable" 
                                                 ref="bulkSelectTable" 
                                                 :data="deviceSnSelection" 
                                                 :showHeader="false"
                                                 :rownumber="false"
                                                 :front-pagination="true"
                                                 height="270px" pagination="true" >
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
                      	
                      	<div v-if="hasRestoreRole" :class="deviceSnSelection.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="importFileBtn">
                     		<span class='el-icon el-icon-operation-import'></span>
                     		<span><%=rb.getString("DaoRuWenJian")%></span>
                     	</div>
                     	<div :class="deviceSnSelection.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="downloadFileBtn">
                     		<span class='el-icon el-icon-operation-export'></span>
                     		<span><%=rb.getString("BackupRestoreDaoChuWenJian")%></span>
                     	</div>
                     </div>
                    <div class='commonFlex'>
                        <el-input placeholder="<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%>" suffic-icon='el-icon-search' v-model="searchText" style="width: 300px; margin: 10px 0 0 20px;">
                            <i slot="suffix" class="el-icon el-icon-common-search"  @click="deviceQuery" style='font-size: 16px; margin-top: 5px;'></i>
                        </el-input>
                        <div class="tableHeadQueryBoxCls">
                            <div v-for="(item,index) in enbAdvancedQueryItemList">
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
                            <div class="advancedQueryItemBox"  style="background: #FFF;" @click="enbClearFilterClick">
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
			<div class="dataStatistics" v-if="activeName == 'taskList'">
				<div class="enableStatistics">		
					<span class="enableTitleStatistics titleCommon"><i class="el-icon el-icon-status-waiting1 statusTip" style=""></i><%=rb.getString("DengDai") %></span>
					<span class="enableNumStatistics titleCommon">{{waitingNum}}</span>
				</div>
				<div class="enableStatistics grantSuspended">		
					<span class="enableTitleStatistics titleCommon"><i class="el-icon el-icon-status-inProgress statusTip"></i><%=rb.getString("JinXingZhong") %></span>
					<span class="enableNumStatistics titleCommon">{{inProgressNum}}</span>
				</div>
				<div class="enableStatistics">
					<span class="enableTitleStatistics titleCommon"><i class="el-icon el-icon-status-suspend statusTip" ></i><%=rb.getString("ZanTing") %></span>
					<span class="enableNumStatistics titleCommon">{{suspendNum}}</span>
				</div>
				<div class="enableStatistics" style="margin-left:10px; border-right: 0;">		
					<span class="enableTitleStatistics titleCommon"><i class="el-icon el-icon-status-terminate statusTip"></i><%=rb.getString("YiJieShu") %></span>
					<span class="enableNumStatistics titleCommon">{{endNum}}</span>
				</div>
			</div>
			<!-- device list 数据统计-->
			<div class="dataStatistics" v-if="activeName == 'deviceList'">
				<div class="enableStatistics">		
					<div class="enableTitleStatistics titleCommon" >
						<i class="el-icon el-icon-circle-success deviceStatusTip"></i>
						<span style="color:#67D972"><%=rb.getString("ChengGong") %></span>
					</div>
					<span class="enableNumStatistics titleCommon">{{successNum}}</span>
				</div>
				<div class="enableStatistics" style="margin: 0 40px 0 10px;border-right: 0">		
					<div class="enableTitleStatistics titleCommon">
						<i class="el-icon el-icon-circle-close deviceStatusTip"></i>
						<span class="redCol"><%=rb.getString("ShiBai") %></span>
					</div>
					<span class="enableNumStatistics titleCommon">{{failNum}}</span>
				</div>
				
				<div class="circleIcon placeholder-bt" style="top: -6px; right: 0;" placeholder="<%=rb.getString("DaoChu")%>">		
					<span class="el-icon el-icon-circle-export" @click="deviceListExport"></span>
				</div>
			</div>
			<!--task list,device list tabs -->
			<el-tabs v-model="activeName" @tab-click='tabClick' style="height:100%;" class='newTabs'>
				<el-tab-pane label="<%=rb.getString("RenWuLieBiao")%>" name="taskList"> 
					<!--task list table -->
			        <el-ctable ref="taskListTable" id="taskListTable" class='commonQueryToolbar'
			        	:time="6"
			        	height="100%"
			        	:url="taskListTableUrl"
			        	:query-params="queryTaskParams"
			        	@load-success="taskListTableLoadSuccess">
			          	<!-- 列表toolbar -->
			          	<template slot="toolbar">
							<div class='toolbarHeadBtnBoxCls commonQuery' style=" padding: 0 0 10px 0;">
								<el-query type="normal" @query="taskListQuery" placeholder="<%=rb.getString("RenWuMingCheng")%>"></el-query>
								<el-date-picker style='margin-left: 10px;' 
									v-model="dateValue"
									type="datetimerange"
									value-format="yyyy-MM-dd HH:mm:ss"
									range-separator="——"  
									@change="dateChange"
									start-placeholder='<%=rb.getString("KaiShiShiJian")%>' 
									end-placeholder='<%=rb.getString("JieShuShiJian")%>'>
								</el-date-picker>
								
								<div class="tableHeadQueryBoxCls">
									<div v-for="(item,index) in advancedQueryItemList">
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
									<div class="advancedQueryItemBox"  style="background: #FFF;" @click="clearFilterClick">
										<%=rb.getString("QingKongShaiXuan")%>
									</div>
								</div>
							</div>
						</template>
						
			          	<!-- 列表columns -->
			          	<el-table-column v-if="hasRestoreRole" prop="op" label=" " width="50" align="center">
			            	<template slot-scope="scope">
			              		<div class="el-icon el-icon-operation-more" @click="optClick(scope.row,event)" v-clickoutside="handerClose"></div>
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
			        <el-cmenu ref="menus" :data="menus" @click="clickMenu"></el-cmenu>				
				</el-tab-pane>
				
				<!--device list table  -->
				<el-tab-pane label="<%=rb.getString("BackupRestoreSheBeiLieBiao") %>" name="deviceList">
					<el-ctable ref="deviceListTable" id="resultTable" class='commonQueryToolbar'
						:url="deviceListTableUrl" 
						:query-params="resultParams" 
						:time="6" 					
						height="100%" 
						@load-success="deviceListTableLoadSuccess">
						<!-- 列表toolbar -->
						<template slot="toolbar">
								<div class='toolbarHeadBtnBoxCls commonQuery' style=" padding: 0 0 10px 0;">
								<el-query type="normal" @query="deviceListQuery" placeholder='<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%>/<%=rb.getString("RenWuMingCheng")%>'></el-query>
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
	<el-slide class='commonBorderSlide' ref="slide"
		:url="slideUrl" 
		:modal='slideModal' 
		:title="slideTitle" 
		:footer="slideFooter" 
		:header='slideHeader' 
		:position="slidePosition"
	 	:height="slideHeight" 
	 	:width="slideWidth" 
        :subloading="slideSubmitLoading"
	 	:ok-text="'<%=rb.getString("QueDing")%>'" 
	 	:cancel-text="'<%=rb.getString("QuXiao")%>'"  
	 	@cancel='cancelSlide' 
	 	@ok='saveSlide'>	 	
	 </el-slide>
	 
	<!-- 导入文件框 -->
	<el-dialog title='<%=rb.getString("PiLiangDaoRu")%>' :visible.sync="showConfirmInfo" width="1100" class="backupRestoreImport"
		:close-on-click-modal="false" @close="closeImport">
		<el-form :model="importForm" ref="importForm" label-position="left" :rules='rules'> 
			<el-form-item label="<%=rb.getString("DaoRuFangShi")%>" label-width="130px">
				<el-radio-group v-model="importModeSelection">
					<el-radio label="moreFile" class="moreFileSelected"><%=rb.getString("AnDaoRuWenJianMingPiPeiSheBei")%></el-radio>
					<el-radio label="singleFile" class="singleFileSelected">
                        <span><%=rb.getString("SuoXuanSheBeiPiPeiYiGePeiZhiWenJian")%></span>
                        <span class="oneFileMoreDevicesTip"><%=rb.getString("ZhiNengXuanZeXiangTongChanPinLeiXingDeSheBei")%></span>
                    </el-radio>
				</el-radio-group>
			</el-form-item>
            <el-form-item label="<%=rb.getString("DaoRuWenJian")%>" label-width="130px" prop="fileName" v-if="importModeSelection == 'moreFile'" class='importSelect'>
	            <el-upload 
	           		ref="moreUpload"
	            	:multiple="true" 	            	
	            	:on-success='moreCheckFile' 
	            	:on-change="moreFileChange"  
	            	:show-file-list=false
				    :action="importForm.uploadFileUrl" 
				    :data="moreFileParams" 
				    name="uploadFile"
				    :file-list="moreFileList" 
				    :http-request="moreFileRequest"
				    :auto-upload="false"
				    :limit="20"
				    :on-exceed="handleExceed"
				    accept=".xml, .nv">
					<el-input :readonly="true" :value=fileName placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>' class="w270" v-if="false">
						<a slot="suffix" class="el-icon el-icon-operation-import importBox" @click="moreFileSelect"></a>						
					</el-input> 	 
					<el-button size="small" type="primary" @click="moreFileSelect"><%=rb.getString("DianJiShangChuan")%></el-button>	
					<span v-if="false" class="fileImportTypeTip"><%=rb.getString("DangQianZhiChixmlGeShi")%></span>	
					<a slot="trigger" ref="file_up"></a>
				</el-upload>	
            </el-form-item>
            
			<el-form-item label="<%=rb.getString("DaoRuWenJian")%>" label-width="130px" prop="fileName" v-if="importModeSelection == 'singleFile'" class='importSelect'>
	            <el-upload 
	            	ref="upload"
	            	:before-upload='beforeUpload' 
	            	:on-success='checkFile' 
	            	:on-change="fileChange"  
	            	:show-file-list=false 	            	
				    :action="importForm.uploadFileUrl" 
				    :data="fileParams" 
				    name="uploadFile" 
				    :auto-upload="false" 
				    accept=".xml,.nv">
					<el-input :readonly="true" :value=fileName placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>' class="w270">
						<a slot="suffix" class="el-icon el-icon-operation-import importBox" @click="fileSelect"></a>						
					</el-input>	
					<span class="fileImportTypeTip"><%=rb.getString("BeiFeiHuiFuDaoRuWenJianLeiXing")%></span>								
					<a slot="trigger" ref="file_up"></a>
				</el-upload>	
            </el-form-item>  
			
			<div style="padding-top:10px;border:1px solid #E9E9E9;" class="selectDeviceTable">
				 <div class='queryGroup'> 
					<el-input v-model="searchValue" class='pairgrid-query' placeholder='<%=rb.getString("JiZhanBianMaJiZhanMingCheng")%>'></el-input>
					<i @click="doFilter" class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
				</div>
				<el-table ref="newTable" :data="tableData.slice((currentPage-1)*pageSize,currentPage*pageSize)" border @sort-change="sortChangeEnb"
					style="height:246px;overflow: auto;flex: auto;margin-top:10px;">
					<el-table-column label="" type="index" min-width="50"></el-table-column>
					<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serial_number" sortable></el-table-column>
					<el-table-column label='<%=rb.getString("HostName")%>' prop="host_name" sortable></el-table-column>
					<el-table-column label='<%=rb.getString("ChanPinLeiXing")%> ' prop="product_type"></el-table-column>
					<el-table-column label='<%=rb.getString("BackupRestorePeiZhiWenJian")%>' prop="file_name" v-if="importModeSelection == 'singleFile'"></el-table-column>
					<el-table-column label='<%=rb.getString("BackupRestorePeiZhiWenJian")%>' prop="file_name" v-if="importModeSelection == 'moreFile'" :formatter="fileNameFormatter"></el-table-column>
					<el-table-column width="1" :show-overflow-tooltip="false">
						<template slot-scope="scope">
							<div slot="reference" class="opts-wrapper" style="overflow: hidden;">
								<i @click="deleteChecked(scope.row,scope.$index)" class="el-icon el-icon-circle-close"> </i>
							</div>
		          		</template>
					</el-table-column>
				</el-table>
				<el-pagination 
					@size-change="handleSizeChange"
					@current-change="handleCurrentChange"
					:current-page="currentPage"
					:page-sizes="[50,100,200]"
					:pageSize="pageSize"
					layout="total,sizes, prev, pager, next, jumper"
					:total="tableData.length">
				</el-pagination>				
			</div>
			
			<div v-show="noSelectDevice" style="color:#FA5555;font-size:12px;"><%=rb.getString("ZhiShaoXuanZeYiGeSheBei")%></div>
			<div v-if="importModeSelection == 'singleFile'">
				<div v-show="noSameProductType" style="color:#FA5555;font-size:12px;"><%=rb.getString("ZhiNengXuanZeXiangTongChanPinLeiXingDeSheBei")%></div>
			</div>			
		</el-form>
		
		<div slot="footer" class="dialog-footer">
			<div class="buttonGroup">
				<el-button type="primary" @click="caCertsUploadImport"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="closeImport"><%=rb.getString("QuXiao")%></el-button>
			</div>	
		</div> 	
	</el-dialog>

</div>
</div>
<script>
	var eNBBackupRestoreTasks = new Vue({
		el:'#backupRestore',
		data(){
			var vm = this,
                validateDeviceFileName = function(rule,value,callback) {
                    var value = vm.fileName,
						productType = vm.tableData[0].product_type;
					//导入方式： 所选设备匹配一个配置文件
                    if(vm.importModeSelection == 'singleFile'){
						if( value === '' || value === null || value === undefined) {
                            callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
                        }else {
							//MLQ, MLN 产品类型只能导入nv文件
                            if(productType == 'MLQ' || productType == 'MLN'){
								if(!fileFormatMatch(value,"nv")){
									callback(new Error("<%=rb.getString("BeiFeiHuiFuDaoRuWenJianLeiXingNV")%>"))
								}else {
									callback();
								}
							}else{
								if(!fileFormatMatch(value,"xml")){
									callback(new Error("<%=rb.getString("DangQianZhiChixmlGeShiAll")%>"))
								}else {
									callback();
								}
							}
                        }
                    }else{
						//导入方式：按照导入文件名匹配设备
                        if( value === '' || value === null || value === undefined) {
                            callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
                        }else {
                            callback();
                        }
                    }
                };
			return{
				periodDataLoading: true,  // 周期备份数据加载状态
				waitingNum:'0',
				inProgressNum:'0',
				suspendNum:'0',
				endNum:'0',				
				successNum:'0',
				failNum:'0',			
				deviceSnSelection:[],
				newDeviceSnSelection:[],
				tableData:[],
				searchValue:"",
				pageSize:50,
				currentPage:1,
				showConfirmInfo:false,
				importModeSelection:'moreFile',
				periodSwitch:'0',
				activeName:'taskList',
				height:'100%',
				deviceUrl:'${ctx}/task/enb/config/backupRestore/queryCellInfos.action',
				//enb devices 参数
				eNBDeviceParams:{
					timeZone: timeZone,
					product_type:'',
					searchText:'',
					likeFields: 'serial_number,host_name'
				},
				filterParams: {
					likeFields: 'serial_number,host_name'
				},
				product_type:'',
				searchText:'',
				
				rowData:[],
				taskListTableUrl: '${ctx}/task/enb/config/backupRestore/queryTaskList.action',
				deviceListTableUrl: '${ctx}/task/enb/config/backupRestore/queryTaskDeviceList.action',
				
		       	queryTaskParams: {
		         	timeZone: timeZone,
		         	searchText: '',	   
		         	likeFields: 'task_name',
					taskName: '',
					createUser: '',
					taskType: '',
					taskStatus: '',
		         	startTime:'',
		         	endTime:''
		       	},
		       	//model
		       	queryForm: {
					taskName:'',
		         	taskStatus:'',
		         	createUser:'',
		         	taskType:'',
		         	timeRange: [],
				},
		       	
		       	//device list  搜索
		       	resultParams: {
		       		timeZone: timeZone,
		       		searchText: '',	
		       		likeFields: 'serial_number,task_name,host_name',
		       	},
		       	menus: [],
		       	
		        fileParams:{},              
	            fileName:'',	            					
				fileList:[],
			
				importForm: {
	                uploadFileUrl: ''
	           	},
	           	
	           	//多设备多文件
	            moreFileParams:{},
			
	            moreFileList:[],
	            fileData:[],

	            onePeriodData:{},
	            
	            rules: {
                    fileName:[
                    	{validator: validateDeviceFileName},
                    ],
	            },
	            noSelectDevice:false,
	            noSameProductType:false,
	            
	            slideUrl:'',
				slideTitle:'',
				slideFooter:'',
				slideHeader:'',
				slidePosition:'',
				slideHeight:'',
				slideWidth:'',
                slideSubmitLoading:'',
				slideModal:'',
				firstOpenImport:false,
				lastOpenImport:false,
				// 当前跳转页面标识
				curSelectPage:'',
				
				bulkSelectShow: false,
				dateValue:[],
				
				advancedQueryItemList:[
					
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

				enbAdvancedQueryItemList:[
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
			isSuperAdmin() {
				return is_super_user == 'true';
			},
			hasRestoreRole() {
				return writableMap.CODE_ENB_BACKUP_RESTORE == true;
			}
		},
		methods:{
		    eNBBackupOrRestoreInit(){
                var vm = this,
                    params={
                        likeFields: 'serial_number,host_name'
                    };

                axios.post( '${ctx}/task/enb/config/backupRestore/getProductType.action', stringify(params)).then(function(response){
                    var data = response.data;
                    
                    var arr = [];
                    (data.product_type || []).map(function(item){
                        if (item){
                            arr.push({label:item.text,value:item.value})
                        }
                    })
                    vm.enbAdvancedQueryItemList.map((items)=>{
                        if('product_type' == items.value){
                            items.options = arr;
                        }
                    })
                });
		    },
			// 获取周期备份任务信息（控制再次打开周期备份任务按钮以及开关的状态）
			getSwitchData(){
				var vm = this;
				
				// 设置加载状态为 true，禁用按钮
				vm.periodDataLoading = true;

				axios.post('${ctx}/task/enb/config/backupRestore/periodTask.action').then(function(response){
					var data = response.data;
					
					if(Object.keys(data).length != 0){
						vm.onePeriodData = data;
						vm.periodSwitch = data.is_enable == 1 ? '1' : '0';
					}else{
						vm.onePeriodData = {};
						vm.periodSwitch = '0';
					}
					
					// 数据加载完成，启用按钮
					vm.periodDataLoading = false;
				}).catch(function(error){
					// 即使失败也要启用按钮，避免永久禁用
					vm.periodDataLoading = false;
				})
			},
			handleExceed(files,fileList){
				this.$message.info('<%=rb.getString("ZuiDuoXuanZeWenJian20")%>');
			},
			
			//导入文件
			importFileBtn(){
				var vm = this;	
				var selectDatas = vm.deviceSnSelection;
				if(selectDatas.length != 0){
					this.showConfirmInfo = true;					
					selectDatas.map((item,index) => {
						var selectRowData = {     						
	      					serial_number:item.serial_number,
	      					host_name:item.host_name,
	      					product_type:item.product_type,
	      					file_name:''     		    			
	      		    	}
						vm.tableData.push(selectRowData);
						vm.newDeviceSnSelection = vm.tableData;
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
			moreCheckFile(res,file){    //发送请求，校验device文件内容 
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
					vm.showConfirmInfo = false;
					vm.$refs.eNBDeviceTable.refresh();
					vm.moreCloseFileSelect();
				}else{
					vm.$message({
						type: 'error',
						message: res.msg
					});
				}
				//修改已选择文件状态  
				var fileList = vm.$refs.moreUpload.uploadFile;
				fileList.forEach(function(file){
					file.status = 'ready';
				})
			},
	    	
			/**
			* 选择文件后，校验格式，并赋值页面显示 
			* @param file{object}   文件信息
			* @param fileList{Array}  文件列表
			*/ 
			moreFileChange(file,fileList){ 
				var vm = this,
					fileName = file.name, //文件名称
					newFileName = fileName.substr(0,fileName.lastIndexOf("_")); // 截取的 '_' 之前的文件名称
				
				//循环tableData，判断product_type
				vm.tableData.map((item,index) => {
					if(item.serial_number == newFileName){
						//item.product_type 为 'MLQ' 或 'MLN' 时，则文件名称为 sn +'_CFG.nv'; 其他情况为 sn+'_CFG.xml';

						if(item.product_type == 'MLQ' || item.product_type == 'MLN'){
							item.file_name = item.serial_number + '_CFG.nv';
						}else{
							item.file_name = item.serial_number + '_CFG.xml';
						}

						vm.fileName += file.name+',';
						vm.moreFileParams.FileName = file.name;
						vm.firstOpenImport = true;
					}
	    	    })
				vm.lastOpenImport = true;
			},
			// 设备组名称格式化
			fileNameFormatter(row,column,cellValue,index){
				var vm = this;
				if(vm.firstOpenImport == false){
					if(vm.lastOpenImport == true){
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
			moreFileSelect(){  
				var vm =this;

				vm.tableData.map((item,index) => {									
					item.file_name = '';										
	    	    }) 
	    	    vm.fileName = '';
	    	    vm.firstOpenImport = false;
				vm.$refs.moreUpload.clearFiles();
				vm.$refs['file_up'].click();
			},
			
			// 移除导入文件
			moreCloseFileSelect(){
				var vm = this;
				vm.fileName = '';			
				vm.$refs.moreUpload.clearFiles();
			},
						
			//提交时匹配的文件及文件流
			moreFileRequest(file){
				var vm = this,
   					newFileName = file.file.name.substr(0,file.file.name.lastIndexOf("_"));// 截取的 '_' 之前的文件名称	

   				vm.tableData.map((item,index) => {				
					if(item.serial_number == newFileName){
						vm.fileData.push(file.file)
					}																	
	    	    })
			},
			
			
			// --------------------------------------多设备导入一个文件--------------------------
		
			/**
			* 文件上传成功函数 
			* @param res{object}   返回信息
			* @param file{object}  文件信息
			*/
			checkFile(res,file){    //发送请求，校验device文件内容 
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
					vm.showConfirmInfo = false;
					vm.$refs.eNBDeviceTable.refresh();
					vm.closeFileSelect();
				}else{
					vm.$message({
						type: 'error',
						message: res.msg
					});
				}
				//修改已选择文件状态  
				var fileList = vm.$refs.upload.uploadFiles;
				fileList.forEach(function(file){
					file.status = 'ready';
				})
			},
	    	
			/**
			* 选择文件后，校验格式，并赋值页面显示 
			* @param file{object}   文件信息
			* @param fileList{Array}  文件列表
			*/ 
			fileChange(file,fileList){ 
				var vm = this;

				vm.fileName = file.name;
				vm.fileParams.fileName = file.name;
				vm.tableData.map((item,index) => {
					//item.product_type 为 'MLQ', 'MLN' 时，则文件名称为 sn +'_CFG.nv'; 其他情况为 sn+'_CFG.xml';
					if(item.product_type == 'MLQ' || item.product_type == 'MLN'){
						item.file_name = item.serial_number + '_CFG.nv';
					}else{
						item.file_name = item.serial_number + '_CFG.xml';
					}
	    	    })

			},
			
			// 选择文件
			fileSelect(){  
				var vm =this;

				vm.tableData.map((item,index) => {									
					item.file_name = '';										
	    	    }) 
	    	    vm.fileName = '';
				vm.$refs.upload.clearFiles();
				vm.$refs['file_up'].click();
			},
			
			// 移除导入文件
			closeFileSelect(){
				var vm = this;
				vm.fileName = '';			
				vm.$refs.upload.clearFiles();
			},
						
			/**
			* 文件上传之前
			* @param file{object}   文件信息
			*/ 
			beforeUpload(file){
				var vm = this,
					fileName = file.name,
					fileSize = file.size,
					fd = new FormData(),
					config = {
						headers: { 'Content-Type': 'multipart/form-data' }
					};
				var data = vm.tableData;
				var cellCodes = '';
				if(data.length != 0) {
					data.map((item,index) => {
						if(item.serial_number != null){
							cellCodes += item.serial_number + ","
						}      				
					})				
				}

				fd.append('uploadFile',file); //文件流
				fd.append('uploadFileName',fileName);//文件名
				fd.append('fileSize',fileSize);//文件名	大小
				fd.append('serial_numbers',cellCodes);//基站编码

				axios.post("${ctx}/task/enb/config/backupRestore/importFile/oneToAny.action",fd,config).then(function(res){
					if(res.data["success"]){	
						vm.$message.success('<%=rb.getString("ChengGong")%>');
						vm.$refs.eNBDeviceTable.refresh();						
						vm.showConfirmInfo = false;
						vm.fileList = [];
						vm.fileName = '';
						vm.$refs.importForm.resetFields();						
					}else{
						vm.$message.error(res.data["message"]);
						vm.showConfirmInfo = false;
					}
				})
				
				return false;
			},
			
			
			/*确定导入*/
	        caCertsUploadImport() {
				var vm = this,
					noSelectData;
				
				if(vm.tableData.length == 0 ){
					vm.noSelectDevice = true;
					noSelectData = false;
				}else{
					vm.noSelectDevice = false;
					noSelectData = true;						
				}
				
				//多设备，单文件
				if(vm.importModeSelection == 'singleFile'){
					var productType = vm.tableData.map(function(item){
						return item.product_type;
					});
					var newProductList = productType.filter(function(item){ 
						return item !== '';
					});
					var lastArr = Array.from(new Set(newProductList)); 
					//只能选择相同产品类型的设备，如果已勾选数据无产品类型时，可忽略此数据，但是给后端传参（serial_numbers）包含这些无产品类型的数据
					if(lastArr.length > 1){
						vm.noSameProductType = true;
						noSelectData = false;
					}else{
						vm.noSameProductType = false;
						noSelectData = true;							
					} 
					
					vm.$refs.importForm.validate((valid) => {
	                    if (valid && noSelectData) {
	                    	vm.$refs.upload.submit();                   	
	                    }
	                }) 
				}else{					
					//多设备  多文件  
					vm.$refs.importForm.validate((valid) => {	
	                    if (valid && noSelectData) {
	    					var fd = new FormData(),
		    					config = {
		    						headers: { 'Content-Type': 'multipart/form-data' }
		    					};
	
		                    vm.$refs.moreUpload.submit();     
		    				var data = vm.tableData;
		    				var cellCodes = '';
		    				if(data.length != 0) {
		    					data.map((item,index) => {
		    						if(item.serial_number != null){
		    							cellCodes += item.serial_number + ","
		    						}      				
		    					})				
		    				}
		    				
		    				var fileNames = '';
		    				vm.fileData.forEach(file=>{			    						
		    					fd.append('uploadFile',file,file.name);
		    					fileNames += file.name + ","
		    				})
		    			
		    				fd.append('uploadFileName',fileNames);//文件名
		    				fd.append('serial_numbers',cellCodes);//基站编码
		    				axios.post("${ctx}/task/enb/config/backupRestore/importFile/anyToAny.action",fd,config).then(function(res){
		    					if(res.data["success"]){	
		    						vm.$message.success('<%=rb.getString("ChengGong")%>');
		    						vm.$refs.eNBDeviceTable.refresh();										
		    					}else{
		    						vm.$message.error(res.data["message"]);
		    					}
		    					vm.closeImport();
		    				})                  	              	
                    	}
                	})
					
				}
					 				
			},
			// 关闭导入弹出框
			closeImport(){
				var vm = this;
				vm.showConfirmInfo = false;	
				vm.fileList = [];
				vm.fileName = '';
				vm.$refs.importForm.resetFields();
				vm.tableData =[];
				vm.noSelectDevice = false;
				vm.noSameProductType = false;
				vm.importModeSelection = 'moreFile';
				vm.fileData =[];
				vm.$refs['eNBDeviceTable'].clearSelection();
				vm.searchValue = '';
				vm.newDeviceSnSelection = [];
				vm.firstOpenImport = false;
				vm.lastOpenImport = false;
			},
							
			//批量导入-表格搜索
			doFilter(){
				var vm = this,
					searchInput = vm.searchValue,
					serchArray = [];

				if(searchInput === '' || searchInput === null || searchInput === undefined){
					vm.updateImportTable(vm.newDeviceSnSelection);
				}else{
					vm.tableData.forEach(function(item){
						if(item.serial_number != null){
							if(item.serial_number.indexOf(searchInput) > -1 || item.host_name.indexOf(searchInput) > -1){
								serchArray.push(item)
							}
						}						
					})
					vm.updateImportTable(serchArray);
				}				
			},
			updateImportTable(arr){
				var vm = this;
				vm.tableData = arr;
				vm.tableData.map((item,index) => {
					item.file_name = item.file_name;	
	    	    })	
				vm.currentPage = 1;
			},
			
			//批量导入-删除
			deleteChecked(row,index){
				var vm = this,
					index = index,
					fileAllName=[];
				if(index >= 0) {
					vm.tableData.splice(index,1);
			    	vm.newDeviceSnSelection.map((item,index) => {
						if(item.serial_number == row.serial_number ){
							vm.newDeviceSnSelection.splice(index,1);
						}			    					    		
			    		//多设备多文件
			    		if(vm.importModeSelection == 'moreFile'){
			    			vm.fileName = '';
							
							if(item.file_name == '' || item.file_name == null || item.file_name == 'undefined'){
								
							}else{
								fileAllName.push(item.file_name)	
							}
				    		vm.fileName = fileAllName.join(',');
						}
			    	})
			    } 			
			},

			handleSizeChange(val){
				this.pageSize = val;
			},
			handleCurrentChange(val){
				this.currentPage = val;
			},
			currentChangePage(list){
				var fromNum = (this.currentPage - 1) * this.pageSize;
				var toNum = this.currentPage * this.pageSize;
				this.tableDataEnd = [];
				for(;fromNum < toNum; fromNum++){
					if(list[fromNum]){
						this.tableData.push(list[fromNum])
					}
				}
			},
					
			//eNB devices 批量下载		
			downloadFileBtn(){
				var vm = this,
					fileNames = '';

				if(vm.deviceSnSelection.length != 0){
					vm.deviceSnSelection.map(function(item){
						if(item.latest_update_file !== '' && item.latest_update_file !== null && item.latest_update_file !== undefined){
							fileNames += item.latest_update_file + ",";							
						}						
					})

					if(fileNames == '' || fileNames == null || fileNames == undefined){
						vm.$message.info('<%=rb.getString("DangQianWuPiPeiWenJian")%>')						
					}else{
						exportByForm("${ctx}/task/enb/config/backupRestore/batch/exportFile.action",{		        	
							fileNames: fileNames
				        });
					}
				}else{
					vm.$message.info('<%=rb.getString("ZhiShaoXuanZeYiGeSheBei")%>')
				} 
	
			},
			//新建备份任务
			backupTaskBtn(){
				var vm = this;
				vm.slideHeader = true;
	    	    vm.slideTitle = '<%=rb.getString("XinJianRenWu")%>';
	    	    vm.slideUrl = '${ctx}/task/profile/backupRestore/goAddBackupRestoreTaskPage.action?taskType=backup';
	    	    vm.slideFooter = 'true';
	    	    vm.slidePosition = 'top';
	    	    vm.slideHeight = '100%';
	    	    vm.slideWidth = '100%';
                vm.slideSubmitLoading = false;
	    	    vm.curSelectPage = 'backup';
	    	    vm.$refs.slide.showSlide(function(){	    	    	
					eventBus.$emit("show-type",'backup','');
	    	    });
			},
			//新建恢复任务
			restoreBtn(){
				var vm = this;
				vm.slideHeader = true;
	    	    vm.slideTitle = '<%=rb.getString("XinJianRenWu")%>';
	    	    vm.slideUrl = '${ctx}/task/profile/backupRestore/goAddBackupRestoreTaskPage.action?taskType=restore';
	    	    vm.slideFooter = 'true';
	    	    vm.slidePosition = 'top';
	    	    vm.slideHeight = '100%';
	    	    vm.slideWidth = '100%';
                vm.slideSubmitLoading = false;
	    	    vm.curSelectPage = 'restore';
	    	    vm.$refs.slide.showSlide(function(){	    	    	
					eventBus.$emit("show-type",'restore','');
	    	    });
			},
			
			//新建周期备份任务
			periodbackupBtn(){
				var vm = this;
				
				// 如果数据还在加载中，不允许点击
				if(vm.periodDataLoading){
					return;
				}
				
				var updateOrAdd = Object.keys(vm.onePeriodData).length;
				
				if(updateOrAdd == 0){
					//新建周期备份任务
					vm.slideHeader = true;
		    	    vm.slideTitle = '<%=rb.getString("XinJianZhouQiBeiFen")%>';
		    	    vm.slideUrl = '${ctx}/task/profile/backupRestore/goAddBackupRestoreTaskPage.action?taskType=backup';
		    	    vm.slideFooter = true;
		    	    vm.slidePosition = 'top';
		    	    vm.slideHeight = '100%';
		    	    vm.slideWidth = '100%';
                    vm.slideSubmitLoading = false;
		    	    vm.curSelectPage = 'periodBackup';
		    	    vm.$refs.slide.showSlide(function(){	    	    	
						eventBus.$emit("show-type",'periodBackup','');
		    	    });
				}else{
					//修改周期备份任务
					vm.slideHeader = true;
		    	    vm.slideTitle = '<%=rb.getString("XiuGaiZhouQiBeiFen")%>';
		    	    vm.slideUrl = '${ctx}/task/profile/backupRestore/goAddBackupRestoreTaskPage.action?taskType=backup';
		    	    vm.slideFooter = true;
		    	    vm.slidePosition = 'top';
		    	    vm.slideHeight = '100%';
		    	    vm.slideWidth = '100%';
                    vm.slideSubmitLoading = false;
		    	    vm.curSelectPage = 'updatePeriodBackup';
		    	    vm.$refs.slide.showSlide(function(){
		    	    	eventBus.$emit("show-type",'updatePeriodBackup',vm.onePeriodData);		    	    	
		    	    });
				}				
			},
			
			//device list 导出
			deviceListExport(){
				var vm =this;
				exportByForm("${ctx}/task/enb/config/backupRestore/exportTaskDeviceList.action",{
					timeZone: timeZone,
					likeFields: 'serial_number,task_name,host_name',
					searchText: vm.resultParams.searchText
				});
			},
			// task list 数据统计
			taskListTableLoadSuccess(data){
				var vm = this;
				if(data.properties){
					vm.waitingNum = data.properties.waiting;
					vm.inProgressNum = data.properties.inProgress;
					vm.suspendNum = data.properties.suspend;
					vm.endNum = data.properties.end; 
				}			
			},
			// device list 数据统计
			deviceListTableLoadSuccess(data){
				var vm = this;
				if(data.properties){
					vm.successNum = data.properties.success;
					vm.failNum = data.properties.fail;
				}				
			},
			
		    optClick(row,ev){
	     		var vm = this,
					status = row.TASK_STATUS;

				vm.taskStatus = row.TASK_STATUS;
				vm.$root.rowData = row;
				vm.menus= [
			          {label:'<%=rb.getString("KaiShi")%>',cls:"CODE_ENB_BACKUP_RESTORE hidden",code:'start'},
			          {label:'<%=rb.getString("ZanTing")%>',cls:"CODE_ENB_BACKUP_RESTORE hidden" ,code:'stop'},
			          {label:'<%=rb.getString("ZhongZhiRenWu")%>',cls:"CODE_ENB_BACKUP_RESTORE hidden",code:'terminate'},
			          {label:'<%=rb.getString("ShanChu")%>',cls:"CODE_ENB_BACKUP_RESTORE hidden",code:'del'}
			    ]
		    			   
		    	initTaskStatus(status,vm.menus);

		    	vm.$nextTick(function(){
		    		document.body.click();
    		    	vm.$refs.menus.show(ev);
		    	});
		    },
	    	
		    /**
			 * 点击操作出现下拉菜单
			 * @param ev: 点击当前项数据是哪个
			 * */
    	    clickMenu(ev){
    	    	var vm = this,
					codes = {
						start:this.startRebootTask,
						stop:this.suspendRebootTask,
						terminate:this.terminateRebootTask,
						del:this.delRebootTask
					}
    	    	if(codes[ev.code]){
    	    		codes[ev.code](vm.$root.rowData["TASK_ID"], vm.$root.rowData["TASK_STATUS"])
    	    	}
    	    },
		    
    	    /**
			 * 开始执行任务
			 * @param task_id: 当前数据ID
			*/
			startRebootTask(task_id){ 
    	    	var vm = this;

    	    	axios.post('${ctx}/task/enb/config/backupRestore/activeTask.action',stringify({
    	    		taskId : task_id,
    	    	})).then(function(response){
    	    		var data = response.data;
    	    		if(data["success"]){
    	    			vm.$message.success('<%=rb.getString("ChengGong")%>');
    	    			vm.$refs.taskListTable.refresh()
    	    		}else{
    	    			vm.$message.error(data["message"])
    	    		}
    	    	})
    	    },
			/**
			 * 暂停任务
			 * @param task_id: 当前数据ID
			*/
    	    suspendRebootTask(task_id){
    	    	var vm = this;
    	    	
    	    	axios.post('${ctx}/task/enb/config/backupRestore/suspendTask.action',stringify({
    	    		taskId : task_id,
    	    	})).then(function(response){
    	    		var data = response.data;

    	    		if(data["success"]){
    	    			vm.$message.success(data["message"]);
    	    			vm.$refs.taskListTable.refresh()
    	    		}else{
    	    			vm.$message.error(data["message"])
    	    		}
    	    	})
    	    },
			/**
			 * 终止任务函数
			 * @param task_id: 当前数据ID
			*/
    	    terminateRebootTask(task_id){ 
    	    	var vm = this;
    	    	axios.post('${ctx}/task/enb/config/backupRestore/terminateTask.action',stringify({
    	    		taskId : task_id,
    	    	})).then(function(response){
    	    		var data = response.data;
    	    		if(data["success"]){
    	    			//vm.$message.success('<%=rb.getString("ChengGong")%>');
    	    			vm.$message.success(data["message"]);
    	    			vm.$refs.taskListTable.refresh()
    	    		}else{
    	    			vm.$message.error(data["message"])
    	    		}
    	    	})
    	    },
			/**
			 *   删除任务
			 * @param task_id: 当前数据ID
			*/
    	    delRebootTask(task_id){
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
    		    			vm.$refs.taskListTable.refresh()
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
		    batchSelect(selection){ 
		    	var vm = this;
		    	vm.deviceSnSelection = selection;
			},
			tabClick() {
				var vm = this;

				vm.$nextTick(function(){
					document.body.click();
				})
			},
						
			// eNB devices 表格搜索
			deviceQuery(){
				var vm = this;

				vm.eNBDeviceParams.searchText = vm.searchText;
			},
			// enb devices 搜索
			deviceListQuery(val) {
	       		var vm = this;
	       		vm.resultParams.searchText = val;
	     	},
	     	// task list  搜索
			taskListQuery(val) {
	       		var vm = this;
				vm.taskListResetQuery();
				Object.assign(vm.queryTaskParams, {
					searchText: val,
					//startTime: '',
					//endTime: ''
				});
	     	},
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
					Object.assign(vm.queryTaskParams, params);
				}else{
					Object.assign(vm.eNBDeviceParams, params);
				}
				
			},
	    	// task list 高级搜索
	     	taskListAdvanceQuery() {
				var vm = this, 
					startTime = '', 
					endTime = '';

				if(vm.queryForm.timeRange && vm.queryForm.timeRange.length) {
				    startTime = vm.queryForm.timeRange[0];
				    endTime = vm.queryForm.timeRange[1];
				}
				
				Object.assign(vm.queryTaskParams,{
				   searchText: '',
				   taskName: vm.queryForm.taskName,
				   createUser: vm.queryForm.createUser,
				   taskType: vm.queryForm.taskType,
				   taskStatus: vm.queryForm.taskStatus,
				   startTime: startTime,
				   endTime: endTime
				})
		    },
			// task list 重置
		    taskListResetQuery() {
				var vm = this;

				Object.assign(vm.queryForm, {
	                taskName: '',
		         	taskStatus:'',
		         	createUser:'',
		         	taskType:'',
		         	timeRange: []
	            });	
	     	},
			//slide 保存
			saveSlide(){
				eventBus.$emit('taskSave-ok')
			},
			//slide 取消
			cancelSlide(){	
				eventBus.$emit('hander-cancel')
			},

			closeBackupRestoreTask(){
				var vm = this;

		    	vm.$refs.slide.hide();
				vm.$refs.eNBDeviceTable.clearSelection();
				vm.bulkSelectShow = false;
		    	vm.$refs.taskListTable.refresh();
		    	vm.$refs.deviceListTable.refresh();
		    	vm.$refs.eNBDeviceTable.refresh();
				//重新获取周期备份任务信息
		    	vm.getSwitchData();
		    },
		 	//点击页面其他地方关闭菜单
	        handerClose(){
				var vm = this;

	            vm.$refs.menus.hide();
	        },
	        sortChangeEnb(data){ //排序点击  --- eNB
				var vm =this;
				vm.currentPage = 1;
				if(data.prop =='serial_number'){
					vm.tableData = vm.tableData.sort(vm.compare(data.prop,data.order == 'ascending'))				
				}else if(data.prop =='host_name'){
					vm.tableData = vm.tableData.sort(vm.compare(data.prop,data.order == 'ascending'))
				}
				vm.tableData = vm.tableData.slice(0,vm.pageSize)
			},
			compare(attr,rev){
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
			
			dateChange(val) {
				var vm = this;
				vm.dateValue = val;
				if(val != null){
					vm.queryTaskParams.startTime = vm.dateValue[0];
					vm.queryTaskParams.endTime = vm.dateValue[1];
				}
			},
			//------------------------------已选
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
				
                vm.$refs["eNBDeviceTable"].clearSelection();
                vm.bulkSelectShow = false;
            },
            // 设备已选表格 单个删除事件
            delBulkSelected(rows){
                var vm = this,
					tabs = 'eNBDeviceTable',
					rowKey = 'serial_number';

				vm.deviceSnSelection = vm.deviceSnSelection.filter((items)=>{
					return items[rowKey] != rows[rowKey]
				});
				var selection = this.$refs[tabs].$refs.ctableInner.store.states.selection,
					irow= selection.filter((items)=>{
						return items[rowKey] == rows[rowKey]
					})[0];
				vm.$refs[tabs].toggleRowSelection(irow,false);
				var idx = vm.$refs[tabs].ckList.indexOf(rows[rowKey]);
				vm.$refs[tabs].ckList.splice(idx,1);
				
                if(vm.deviceSnSelection.length == 0){
                	vm.bulkSelectShow = false;
                }
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
    			Object.assign(vm.queryTaskParams, params);
    			document.body.click();
    		},
            // 清除筛选
            enbClearFilterClick(){
                var vm = this,
                    params = {};
                vm.enbAdvancedQueryItemList.map((items)=>{
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
                Object.assign(vm.eNBDeviceParams, params);
                document.body.click();
            },
		},
		watch:{
			importModeSelection(val){
				var vm = this;

				vm.fileData =[];
				vm.searchValue = '';
				vm.fileName = '';
				vm.tableData.map((item,index) => {				
					item.file_name = '';										
	    	    })	
	    	    vm.firstOpenImport = false;
				vm.lastOpenImport = false;
	    	    vm.updateImportTable(vm.newDeviceSnSelection);
			},
			dateValue(newVal){
				var vm = this;
				if(!newVal){
					newVal = [];
					vm.queryTaskParams.startTime = '';
	    			vm.queryTaskParams.endTime = '';
				}
			},
		},
		mounted(){
			this.getSwitchData();
		    this.eNBBackupOrRestoreInit();
			eventBus.$off("cancel-backupRestore").$on('cancel-backupRestore',this.closeBackupRestoreTask);
		}
	})
</script>