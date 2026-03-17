<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>
<style>
	#kpiExport .el-tabs--top { height: 100%; }
	#kpiExport .el-table__header-wrapper { display: block; }
	#kpiExport .el-table--border,.el-pagination { border: none; }
	#kpiExport .el-table td { padding: 20px 0; border: none; }
	#kpiExport .el-table--striped, #kpiExport .el-table__body tr.current-row>td, #kpiExport .el-table tr:hover>td, #kpiExport .el-table--striped .el-table__body tr.el-table__row--striped td { background: #fff; }
	.el-progress-bar { padding-right: 0; }
	.el-progress__text { display: none; }
	.btnStyle { vertical-align: middle; text-align: center; display: inline-block; width: 83px; height: 21px; line-height: 21px; cursor: pointer; margin-left: 18px; border-radius: 2px; }
	.deleteBtn { color: #1DA3FC; border: 1px solid #1DA3FC; }
	.enableBtnSty { background: #1DA3FC; color: #ffffff; }
	.disEnableBtnSty { background: #b0cbdd; color: #ffffff; }
	.el-icon-circle-close { color: #f56c6c; }
	#kpiExport .el-tab-pane { height: 93%; width: 100%;flex: 1 1 100%; display: flex; flex-direction: column; }
	#kpiExport .el-bulk .selected-items-info { width: 760px;}
	#kpiExport .el-bulk .selected-items-info .list-body .list-item-info { width: 740px; text-overflow: ellipsis; white-space: nowrap; overflow: hidden;}
	#kpiExport .el-bulk .selected-items-info .list-item-txt { width: 700px;}
	
</style>
<!--导出页面  -->
<div class='panelDefault commonWarp' style='margin: 0px; width: calc(100% - 0px); height: calc(100% - 0px);overflow:hidden; background: #FFFFFF;' id="kpiExport">
	<div class="circleIcon">
		<span class="el-icon el-icon-close" @click="closeExportPage"></span>
		<div class="titleButtonText"><%=rb.getString("GuanBi")%></div>
	</div>
	<el-tabs v-model="activeName" @tab-click='handleClick' class='commonTabsTop'>
		<!-- 手动生成 -->
		<el-tab-pane label="<%=rb.getString("ShengChenBaoBiao")%>" name="first" style='background: #FFFFFF;'>
			<div style="margin-left:35px;height:60px;line-height:60px;" class="CODE_PERFORMANCE_VIEW hidden">
				<div style='display:inline-block;'><%=rb.getString("DangQianChaXunMuBan")%>" <span>{{exportTemplateName}}</span>"<%=rb.getString("DouHao")%> <%=rb.getString("ChaXunLIDuWei")%>" <span>{{exportSelectReport}} </span>"</div>
				<div style="display:inline-block; margin-left: 20px;"><%=rb.getString("GenJuChaXunJieGuoShengChengBaoBiao")%></div>
				<span class="el-button el-button--primary" @click="generateFile()" style='margin-left: 20px;'><%=rb.getString("ShengCheng")%></span>
			</div>
			<el-progress-table :id="'generate_table'" ref="generate" :url="enbGnbTableUrl" :network-type="exportNetType" :type="'generate'" @func='firstData'></el-progress-table>
		</el-tab-pane>
		<!-- 定时报表 -->
		<el-tab-pane label="<%=rb.getString("DingShiBaoBiao")%>" name="second" style='background: #FFFFFF;'>	
			<div style='display:inline-block; padding: 20px 35px;'><%=rb.getString("DangQianChaXunMuBan")%>" <span>{{exportTemplateName}}</span>"<%=rb.getString("DouHao")%> <%=rb.getString("ChaXunLIDuWei")%>" <span>{{exportSelectReport}} </span>"<%=rb.getString("DouHao")%></div>
			<el-progress-table :id="'regular_table'" ref="timed" :url="enbGnbRegularTableUrl" :network-type="exportNetType" :type="'timed'" @func='firstData'></el-progress-table>
		</el-tab-pane>
	</el-tabs>
	<!--批量组件弹窗-->
	<el-bulk ref="generateBulk" 
		:target="tableTarget" 
		:list="batchSelectData" 
		row-key="id" show-prop="fileName" class="el-bulk"
		:message="{title:'<%=rb.getString("YiXuanWenJian")%>',subTitle:'<%=rb.getString("WenJianMing")%>',clear:'<%=rb.getString("QingKong")%>',cancel:'<%=rb.getString("QuXiao")%>'}">
		<template slot="button">				
			<a class="linkbutton" @click="batchDeleteClick"><span><%=rb.getString("PiLiangShanChu")%></span></a>
		</template>
	</el-bulk>
 
</div>
<template id="tableProgress">
	<el-ctable :id="tableId" ref="ctable" :url="url" time="6" :pagination=true :query-params="taskParams" :row-key="'id'"
		@selection-change='batchSelect'
		height="100%" style="margin: 0 35px;overflow: auto;">

		<template slot="toolbar">
			<div class="queryGroup">
				<el-input class='pairgrid-query' style='width:400px;' v-model="taskParams.searchText" @keyup.enter.native="searchResult"
					placeholder="<%=rb.getString("WenJianMing")%>" size="mini" ></el-input>
		    	<i @click='searchResult' class="el-icon el-icon-common-search"></i>
			</div>        	
         </template>
		<el-table-column type="selection" :reserve-selection="true" v-if="writableMap['CODE_PERFORMANCE_VIEW'] == true && batchOperation == true"></el-table-column>
		<el-table-column width="80">
			<template slot-scope="scope">
				<div class="tableDiv operation_zip"></div>
			</template>
		</el-table-column>
		<el-table-column prop="id" v-if="false"></el-table-column>
		<el-table-column prop="fileName" v-if="false" ></el-table-column>
		<el-table-column label="<%=rb.getString("WenJianMing")%>" prop="fileName">
			<template slot-scope="scope">
				<div>{{scope.row.fileName}}</div>
				
			</template>
		</el-table-column>
		<el-table-column  label="<%=rb.getString("JinDu")%>" prop="fileProgress" width="220">
			<template slot-scope="scope">
				<div class='commonFlex'>
					<el-progress :percentage="scope.row.fileProgress" :status="scope.row.result ==0?'exception':''" stroke-width="8" show-text=false style='width: 100px; margin: 8px 24px 0 0;'></el-progress>
					<div v-if="scope.row.result == 0" style="color:#f36666;font-size:14px;">{{scope.row.fileProgress}}%</div>
					<div v-else-if="scope.row.result == 1" style="color:#c2c2c2;font-size:14px;" >{{scope.row.fileProgress}}%</div>
					<div v-else style="color:#1da3fc;font-size:14px;" >{{scope.row.fileProgress}}%</div>
				</div>
			</template>
		</el-table-column>
		<el-table-column label="<%=rb.getString("ZhuangTai")%>" prop="result" width="120">
			<template slot-scope="scope">
				<div v-if="scope.row.result == 0" style="color:#f36666;font-size:14px;"><%=rb.getString("ShiBai")%></div>
				<div v-else-if="scope.row.result == 1" style="color:#c2c2c2;font-size:14px;" ><%=rb.getString("DengDai")%></div>
				<div v-else-if="scope.row.result == 2" style="color:#1da3fc;font-size:14px;" ><%=rb.getString("JinXingZhong")%></div>
				<div v-else style="color:#1da3fc;font-size:14px;" ><%=rb.getString("WanCheng")%></div>
			</template>
		</el-table-column>
		<el-table-column label="" width="100">
			<template slot-scope="scope">
				<div v-if="scope.row.result == 3" class="el-icon el-icon-operation-download" @click="downloadFile(scope.row,event)" style='margin-right: 20px;'></div>
				<div v-if="scope.row.result == 2" v-show="viewShow" class="el-icon el-icon-operation-awaiting" @click="cancelDownload(scope.row,event)" style='margin-right: 20px;'></div>
				<div v-else v-show="viewShow" class="el-icon el-icon-operation-delete" @click="deleteFile(scope.row,event)"></div>
			</template>
		</el-table-column>
	</el-ctable>
</template>
<form id="downLoadKPIFile" style="display:none" method="post" action=""></form>

<script>
	var curEnbGnbEgwType = sessionStorage.getItem('exportEnbGnbOrEgw').split(',');
	var curTemplateName = curEnbGnbEgwType[0], curSelectReport = curEnbGnbEgwType[1];
	
	closeLoading();
	
	var kpiExport = new Vue({
		el:"#kpiExport",
		data(){
			var curForm = kpiQueryVue.tplTabForms.filter((item) => { return item.tempId == kpiQueryVue.templateTabsValue })[0];

			return {
				fileId: '',
				params:{	
					timeZone: timeZone,			
					tempId: curForm.tempId,
		        	searchText: curForm.searchText,
					groupId: isAllOperatorTemp?'':curForm.deviceGroupParam,
		            serialNumber: '', //无用参数
		            hostName: '', //无用参数 
		            startTime: '',
		            endTime: '',
		            reportPeriod: curForm.periodActive,
		            operatorCode: isAllOperatorTemp?curForm.operatorParam:''
				},
				activeName: 'first',
				enbGnbTableUrl: '',
				enbGnbRegularTableUrl: '',
				
				batchSelectData:[],
				tableTarget:'generate_table',
				bulkSelectShow: false,
				exportTemplateName: curTemplateName,
				exportSelectReport: curSelectReport,
			}
		},
		components:{
			'el-progress-table': {
				template: '#tableProgress',
				
				props:{
					url : String,
					type: String,
					id: String,
					networkType: String 
				},	
				
				data(){
					return {
						taskParams: {
							searchText: '',
							type: this.type
						},
						tableId: this.id,
						subBatchSelectData:[]
					}
				},
				computed: {
					viewShow() {
						return writableMap['CODE_PERFORMANCE_VIEW'] == true;						
					},
					// 直接使用父组件传递的 networkType prop
					exportNetType() {
						return this.networkType ? this.networkType : sysMain.headType;
					}
				},
				
				methods:{
					//0-4G, 1-5G, 2-eGW
					searchResult(){
						this.$refs.ctable.refresh();
					},	
					batchSelect(selection){
						var vm = this;
						vm.subBatchSelectData = selection.map((item)=>{
							return Object.assign(item,{fileName:item.fileName});					
						});
						vm.$emit('func', vm.subBatchSelectData)
					},
					
					cancelDownload(row,event){
						var vm = this, curTerminateUrl = '';
						
						if( vm.exportNetType == 'enb'){
							curTerminateUrl = '${ctx}/pm/template/export/terminateManualExportTask.action';
						}else if( vm.exportNetType == 'gnb'){
							curTerminateUrl = '${ctx}/gnb/pm/template/export/terminateManualExportTask.action';
						}else if( vm.exportNetType == 'egw'){
							curTerminateUrl = '${ctx}/egw/pm/template/export/terminateManualExportTask.action';
						}
						
						this.$confirm('<%=rb.getString("QueDingTingZhiShengChengBaoBiao")%>','<%=rb.getString("QueRen")%>',{
							customClass: 'warningConfirm',
							confirmButtonText: '<%=rb.getString("QueDing")%>',
							cancelButtonText: '<%=rb.getString("QuXiao")%>',
							type: 'warning',
							closeOnClickModal: false
						}).then(function(){											
							axios.post(curTerminateUrl,stringify({id : row.id,})).then(function(response){
			    				let data = response.data;
			    				
			    				if(data.success){
			    					vm.$refs.ctable.refresh();
			    					vm.$message({
		    			    			type: 'success',
		    			    			message: '<%=rb.getString("ChengGong")%>'
		    			    		})
			    				}else
			    					vm.$message.error(data["message"]);
			    			}).catch(function(error){})
						}).catch(function(){})
					},
					
					deleteFile(row,event){
						var vm = this, curDeleteFileUrl = '';
						
						if( vm.exportNetType == 'enb'){
							curDeleteFileUrl = '${ctx}/pm/template/export/delManualDataFile.action';
						}else if( vm.exportNetType== 'gnb'){
							curDeleteFileUrl = '${ctx}/gnb/pm/template/export/delManualDataFile.action';
						}else if( vm.exportNetType == 'egw'){
							curDeleteFileUrl = '${ctx}/egw/pm/template/export/delManualDataFile.action';
						}
						
						this.$confirm('<%=rb.getString("QueRenShanChuWenJian")%>','<%=rb.getString("QueRen")%>',{
							customClass: 'warningConfirm',
							confirmButtonText: '<%=rb.getString("QueDing")%>',
							cancelButtonText: '<%=rb.getString("QuXiao")%>',
							type: 'warning',
							closeOnClickModal: false
						}).then(function(){											
							axios.post(curDeleteFileUrl,stringify({id: row.id, type: vm.type || 'generate'})).then(function(response){			    				
								let data = response.data;
								
			    				if(data.success){
			    					vm.$refs.ctable.refresh();
			    					vm.$message({
		    			    			type: 'success',
		    			    			message: '<%=rb.getString("ChengGong")%>'
		    			    		})
			    				}else
			    					vm.$message.error(data["message"]);
			    			}).catch(function(error){})
						}).catch(function(){})
					},
					
					downloadFile(row,event){
						var vm = this, curDownloadFileUrl = '', curExportUrl= ''; 
						if( vm.exportNetType == 'enb'){
							curDownloadFileUrl = '${ctx}/pm/template/export/checkFileIsExist.action';
							curExportUrl = '${ctx}/pm/template/export/downloadManualDataFile.action';
						}else if( vm.exportNetType == 'gnb'){
							curDownloadFileUrl = '${ctx}/gnb/pm/template/export/checkFileIsExist.action';
							curExportUrl = '${ctx}/gnb/pm/template/export/downloadManualDataFile.action';
						}else if( vm.exportNetType == 'egw'){
							curDownloadFileUrl = '${ctx}/egw/pm/template/export/checkFileIsExist.action';
							curExportUrl = '${ctx}/egw/pm/template/export/downloadManualDataFile.action';
						}
						
						axios.post(curDownloadFileUrl,stringify({id: row.id, type: vm.type || 'generate'})).then(function(response){
		    				let data = response.data;
		    				
		    				if(data.success){
		    					exportByForm(curExportUrl,{
		    						id: row.id,
		    						type: vm.type || 'generate'
		    					});
		    				}else
		    					vm.$message.error(data["message"]);
		    			}).catch(function(error){})
					},
				}
			},
			
		},
		computed: {
			exportNetType() {
				return kpiQueryVue.currentNetworkType ? kpiQueryVue.currentNetworkType : sysMain.headType;
			}
		},
		methods:{
			//0-4G, 1-5G, 2-eGW
			init(){
				var vm = this;
				
				if( vm.exportNetType == 'enb'){
					vm.enbGnbTableUrl = '${ctx}/pm/template/export/getManualTaskListPageData.action';
					vm.enbGnbRegularTableUrl = '${ctx}/pm/template/export/getTimingTaskListPageData.action';
				}else if( vm.exportNetType == 'gnb'){
					vm.enbGnbTableUrl = '${ctx}/gnb/pm/template/export/getManualTaskListPageData.action';
					vm.enbGnbRegularTableUrl = '${ctx}/gnb/pm/template/export/getTimingTaskListPageData.action';
				}else if( vm.exportNetType == 'egw'){
					vm.enbGnbTableUrl = '${ctx}/egw/pm/template/export/getManualTaskListPageData.action';
					vm.enbGnbRegularTableUrl = '${ctx}/egw/pm/template/export/getTimingTaskListPageData.action';
				}
			},
			firstData(data){
				var vm = this;
				vm.batchSelectData = data;				
			},
			batchDeleteClick(){
				var vm = this, curBatchDelUrl = '', ids=[], curDelType = '';
					ids = vm.batchSelectData.map((item,index) => {
						return item.id;
					})
				if(vm.activeName == 'first'){
					curDelType = 'generate';
				}else{
					curDelType = 'timed';
				}
				
				var vm = this, curDeleteFileUrl = '';
				
				if( vm.exportNetType == 'enb'){
					curBatchDelUrl = '${ctx}/pm/template/export/delPmReportFile.action';
				}else if( vm.exportNetType == 'gnb'){
					curBatchDelUrl = '${ctx}/gnb/pm/template/export/delPmReportFile.action';
				}else if( vm.exportNetType == 'egw'){
					curBatchDelUrl = '${ctx}/egw/pm/template/export/delPmReportFile.action';
				}
				
				this.$confirm('<%=rb.getString("QueRenShanChuWenJian")%>','<%=rb.getString("QueRen")%>',{
					customClass: 'warningConfirm',
					confirmButtonText: '<%=rb.getString("QueDing")%>',
					cancelButtonText: '<%=rb.getString("QuXiao")%>',
					type: 'warning',
					closeOnClickModal: false
				}).then(function(){											
					axios.post(curBatchDelUrl,stringify({
						id: ids.join(','), 
						type: curDelType
					})).then(function(response){			    				
						let data = response.data;
						
	    				if(data.success){
	    					if(vm.activeName == 'first'){
	    						vm.$refs.generate.$refs.ctable.refresh();
	    					}else{
	    						vm.$refs.timed.$refs.ctable.refresh();
	    					}
	    					vm.$message({
    			    			type: 'success',
    			    			message: '<%=rb.getString("ChengGong")%>'
    			    		})
	    				}else{
	    					vm.$message.error(data["message"]);
	    				}
	    				vm.batchSelectData = [];	
	    				if(vm.activeName == 'first'){
	    					vm.$refs.generate.$refs.ctable.clearSelection();
	    				}else{
	    					vm.$refs.timed.$refs.ctable.clearSelection();
	    				}
	    			}).catch(function(error){})
				}).catch(function(){})
				
			},
			generateFile(){
				var vm = this, curGenerateFileUrl = '';
				
				if( vm.exportNetType == 'enb'){
					curGenerateFileUrl = '${ctx}/pm/template/export/addManualExportTask.action';
				}else if( vm.exportNetType == 'gnb'){
					curGenerateFileUrl = '${ctx}/gnb/pm/template/export/addManualExportTask.action';
				}else if( vm.exportNetType== 'egw'){
					curGenerateFileUrl = '${ctx}/egw/pm/template/export/addManualExportTask.action';
				}

				var curForm = kpiQueryVue.tplTabForms.filter((item) => { return item.tempId == kpiQueryVue.templateTabsValue })[0];
				//不区分网元，不根据时间轴选择一天日期，只根据当前日期选择框的开始时间，结束时间查询
				vm.params.startTime = curForm.dateValue[0] + ' 00:00:00';
				var endTime = curForm.dateValue[1] + ' 00:00:00';
				vm.params.endTime = dateformatter(addDate(new Date(endTime),1));
				
				axios.post(curGenerateFileUrl, stringify(vm.params)).then(function(response){
						let data = response.data;
						
						if(data["success"]){
	   	   					vm.$message({
	    						message: '<%=rb.getString("ShengChengBaoBiaoRenWuYiJianLi")%>'+data["message"],
	    						type: 'success',
	    						onClose: function(){
	    							vm.$refs.generate.$refs.ctable.refresh();
	    						}
	    					})
						} else {   							
							vm.$message.error(data["message"]);
						}
					}).catch(function(error){})
			},
			
			handleClick(tab,event){
				var vm = this;
				vm.batchSelectData = [];
				if(vm.activeName == 'first'){
					vm.tableTarget = 'generate_table';
					vm.$refs.generate.$refs.ctable.clearSelection();
				}else{
					vm.tableTarget = 'regular_table';
					vm.$refs.timed.$refs.ctable.clearSelection();
				}
			},
			
			closeExportPage(){
				$("#exportKpiDiv").slideUp(500,function(){
					$("#exportKpiDiv").html("");
				});
			}
		},
		mounted(){
			this.init();
		}
	});
</script>	