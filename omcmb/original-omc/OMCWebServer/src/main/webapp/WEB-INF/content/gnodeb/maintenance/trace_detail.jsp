<%@ page import="java.util.Locale"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<!DOCTYPE html>
<html>
<head>
	<title>Trace</title>
    <style>
    	.overflow-cls {
			display: inherit;
		}
  	</style>
</head>
<body>
	<!-- gnb信令追踪 -->
	<div class="overflow-cls">
		<div id="tracePage_gnb" class="container commonWarp leftWarp" style="min-width: 900px;position: relative;">
	        <!-- 操作按钮 -->
	        <div v-if="isAdmin" class="circleIcon placeholder-bt" placeholder="<%=rb.getString("TianJia")%>" @click="openChoseTraceTypeTask" style='top: 50px;'>
            	<span class="el-icon el-icon-circle-add"></span>
          	</div>
          	<span class='commonText14 leftBoxHeader'>{{listTitle}}</span>
        	<!-- 表格组件 -->
        	<el-ctable ref="tableTraceList" id="traceList_gnb" :url="traceUrl" :query-params="queryParams" :time="6">
          		<template slot="toolbar">
					<div class='toolbarHeadBtnBoxCls commonQuery' style="height:40px; padding: 0 0 10px 0;">
						<el-query type="normal" @query="query" placeholder="<%=rb.getString("GenZongMingCheng")%>/<%=rb.getString("GenZongCanKaoHao")%>/<%=rb.getString("ShiBieXinXi")%>"></el-query>
						<el-date-picker style='margin-left: 10px;' 
							v-model="dateValue"
							type="datetimerange"
							value-format="yyyy-MM-dd HH:mm:ss"
							range-separator="——"  
							@change="dateChange"
							start-placeholder='<%=rb.getString("KaiShiShiJian")%>' 
							end-placeholder='<%=rb.getString("JieShuShiJian")%>'>
						</el-date-picker>
                        <el-popfilter style="margin: 0 10px;"
                            label='<%=rb.getString("ZhuangTai")%>'
                            v-model="queryParams.taskStatus"
                            type="single"
                            :list="taskStatusList">
                        </el-popfilter>
					</div>
                </template>					
          		<!-- 列表columns -->
				<el-table-column prop="op" label=" " width="50" align="center">
	            	<template slot-scope="scope">
	              		<div class="el-icon el-icon-operation-more" v-clickoutside="hideMenus" @click="optClick(scope.row,event)"></div>
	            	</template>
          		</el-table-column>
          		<el-table-column label='<%=rb.getString("GenZongCanKaoHao")%>' width="100" prop="trace_id" sortable></el-table-column>
          		<el-table-column label='<%=rb.getString("GenZongMingCheng")%>' min-width="280" prop="trace_name" sortable></el-table-column>
          		<el-table-column label='<%=rb.getString("JieKouLeiXing")%>' width="200" prop="ne_interface"></el-table-column>
          		<el-table-column label='<%=rb.getString("ShiBieXinXi")%>' width="220" prop="identification_info"></el-table-column>
		        <el-table-column label='<%=rb.getString("ZhuangTai")%>' width="150" prop="task_status">
					<template slot-scope="scope">
						<div v-if="scope.row.task_status == '1'">
              				<span class="el-icon el-icon-status-waiting1 curStatus"></span>
							<span><%=rb.getString("DengDai")%></span>
						</div>
						<div v-if="scope.row.task_status == '2'">
							<span class="el-icon el-icon-status-inProgress curStatus"></span>
							<span><%=rb.getString("JinXingZhong")%></span>
						</div>
						<div v-if="scope.row.task_status == '3'">
							<span class="el-icon el-icon-status-terminate curStatus"></span>
							<span><%=rb.getString("YiJieShu")%></span>
						</div>
            		</template>		        	
		        </el-table-column>
		        <el-table-column label='<%=rb.getString("KaiShiShiJian")%>' width="180" prop="start_time" sortable></el-table-column>
		        <el-table-column label='<%=rb.getString("JieShuShiJian")%>' width="180" prop="end_time" sortable></el-table-column>
		        <el-table-column label='<%=rb.getString("ChuangJianZhe")%>' prop="create_user" width="180"></el-table-column>
           		<el-table-column label='<%=rb.getString("ChuangJianShiJian")%>' prop="create_time" width="180" sortable></el-table-column>
        	</el-ctable>
        	<el-cmenu ref="menu"  @click="menuClick" :data="menus"></el-cmenu>

		    <el-slide ref="resultSlide" position="bottom" :footer="false" height="300" @cancel="cancelResultSlide" :title="resultTitle">
				
				<template slot='toolbar'>
			 		<a href='#' @click='exportSignaling' class='el-icon el-icon-operation-export exportBtn' style='top: 0; right: 60px;'></a>
			 	</template>
				<el-ctable ref="resultList" id="resultTable_gnb" :url="resultUrl" :query-params="resultParams" :time="6">
					<el-table-column prop="interface" label="<%=rb.getString("JieKouLeiXing")%>"></el-table-column>
					<el-table-column prop="direction" label="<%=rb.getString("FangXiang")%>"></el-table-column>
					<el-table-column prop="protocol" label="<%=rb.getString("XieYi")%>"></el-table-column>
					<el-table-column prop="message" label="<%=rb.getString("XiaoXi")%>">
						<template slot-scope="scope">
							<div>
								<a style='color:#1DA3FC;cursor:pointer;' @click='openTrace(scope.row.id)'>{{scope.row.message}}</a>
							</div>
						</template>
					</el-table-column>
					<el-table-column prop="report_time_sec" label="<%=rb.getString("ShiJian")%>">
						<template slot-scope="scope">
							<div>
								<span>{{scope.row.report_time_sec}}.{{scope.row.report_time_ns}}</span>
							</div>
						</template>
					</el-table-column>
				</el-ctable>
			</el-slide>
			<!-- 新建，详情跳转 -->
		    <el-slide ref="addTrace" :title="slideTitle" method='get'
			    :url="traceAddUrl"  
			    :footer="slideFooter" 
			    :header='slideHeader' 
			    :position="slidePosition"
			    :height="slideHeight" :modal='modal' 
			    :width="slideWidth" 
                :subloading="slideSubmitLoading"
			    @ok='saveTrace'
			    @cancel='cancelAddTrace' 
			    :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'">
				
			 </el-slide>
		</div>
	</div>
	<div id="winTest_gnb" class="easyui-window" title="<%=rb.getString("XinLingXiaoXi")%>"
	     data-options="modal:true,closed:true,collapsible:false,minimizable:false,maximizable:true,width:1050,height:600,resizable:true,inline:false,draggable:true">
	</div>
	<script>
    	//var refreshTaskTimer;
	  	var traceVueGnb = new Vue({
		  	el: '#tracePage_gnb',
			data () {
				return {
	          	 	traceUrl: '${ctx}/signaling/querySignalingTraceTaskList.action',
	         	 	
					queryForm: {
						traceId: '',
						traceName: '',
						identificationInfo: '',
						taskStatus: '',						
	            		timeRange: []
	          		},
		            queryParams: {
			            traceId: '',
						traceName: '',
						identificationInfo: '',
						taskStatus: '',			
			            startTime: '',
			            endTime: '',
			            timeZone: timeZone,
			            searchText: '',
			            likeFileds: 'trace_id,trace_name,identification_info',
			            deviceType: '${ type }',
		            },
		            resultTitle:'<%=rb.getString("JieGuo")%>',
		            resultUrl: '',
		          	resultParams: {
		          		traceId: '',
			            timeZone: timeZone,
		          	},
		          	traceAddUrl: '',
	         		menus: [],
	         		rowData: [],
	         		//slide
	         		slideTitle:'',
	         		slideHeader:'',
	        	    slideFooter:'',
	        	    slidePosition:'',
	        	    slideHeight:'',
	        	    slideWidth:'',
                    slideSubmitLoading:'',
	        	    operateType:'',
	        	    
	        	    dateValue: [],
                    taskStatusList:[
                        {value:'',label:'<%=rb.getString("QuanBu")%>'},
                        {value:'1',label:'<%=rb.getString("DengDai")%>'},
                        {value:'2',label:'<%=rb.getString("JinXingZhong")%>'},
                        {value:'3',label:'<%=rb.getString("YiJieShu")%>'}
                    ],
	        	    advancedQueryItemList:[
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
								{value:'3',label:'<%=rb.getString("YiJieShu")%>'}
							],
							value:'taskStatus',
						}
					],
		      	}
		    },
			computed: {
				listTitle() {
					var title = '<%=rb.getString("eNBXinLingZhuiZong")%>';

					title = title.replace('eNB','gNB');
					
					return title;
				},
				isAdmin(){
					return is_super_user == 'true';
				}
			},
	    	methods: {
				optClick(row,ev){
			    	var vm = this, startFlag = false, terminateFlag = false, restartFlag = false, delFlag = false, 
			    		status = row.task_status;
			    	
	    		    vm.rowData = row;
	    		  	//status 1-等待(可开始，可删除)， 2-进行中(可终止，不可删除)，3 已结束 (可重新开始，可删除)
	    		    if(status == '1'){
	    		    	startFlag = true;
	    		    	terminateFlag= false;
	    		    	restartFlag= false;
	    		    	delFlag = true;
		        	}else if(status == '2'){
		        		startFlag = false;
		        		terminateFlag= true;
	    		    	restartFlag= false;
	    		    	delFlag = false;
		        	}else{
		        		startFlag = false;
		        		terminateFlag= false;
	    		    	restartFlag= true;
	    		    	delFlag = true;
		        	}
    		    	vm.menus= [
    			          {label:'<%=rb.getString("JieGuo")%>',cls:"el-icon el-icon-operation-result",code:'result'},
    			          {label:'<%=rb.getString("KaiShi")%>',cls:"el-icon el-icon-operation-start",code:'start',show: startFlag && vm.isAdmin},
    			          {label:'<%=rb.getString("ZhongZhi")%>',cls:"el-icon el-icon-operation-terminate",code:'terminate', show: terminateFlag && vm.isAdmin},
    			          {label:'<%=rb.getString("ChongXinKaiShi")%>',cls:"el-icon el-icon-operation-restart",code:'reStart', show: restartFlag && vm.isAdmin},
    			          {label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'info'},
    			          {label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete",code:'del',disable:!delFlag && vm.isAdmin}
    			    ]
    		    	this.$nextTick(function(){
    		    		document.body.click();
	    		    	vm.$refs.menu.show(ev);
    		    	});
		        },
	     
				menuClick(ev) {
			        var vm = this,
		              	codes = {
			        		result: vm.resultTrace,
		    	    		start: vm.startTrace,
		    	    		terminate: vm.terminateTrace,
		    	    		reStart: vm.restartTrace,
			    	    	info: vm.infoTrance,
			    	    	del: vm.deleteTrace
		              	};
		
			        if(codes[ev.code]){
			    		codes[ev.code](this.rowData)
			    	}
				},
				
				//结果
				resultTrace(row) {
		            var vm = this;
					vm.resultTitle = "<%=rb.getString("JieGuo")%> (<%=rb.getString("GenZongCanKaoHao")%>:" + row.trace_id + ")";
		            Object.assign(vm.resultParams,{
		            	traceId: row.trace_id
		            });
		            vm.resultUrl = '${ctx}/signaling/querySignalingInfoList.action?type=gNB';
		        	vm.$refs.resultSlide.showSlide();
				},
				//结果-导出
				exportSignaling(){
					var vm = this;
					var params ={
                        timeZone: timeZone,
						traceId : vm.rowData.trace_id
					}
					axios.post('${ctx}/signaling/createSignalingPcapFile.action',stringify(params)).then(function(response){
						var data = response.data;
			            if(data["success"]){
			            	exportByForm("${ctx}/signaling/exportSignalingPcapFile.action",params);
			            }else{
			              	vm.$message.error(data["message"])
			            }
			    	})
				},
				//message 点击 查看详细信令   二进制  树相互转换
				openTrace(id){
					var vm = this;
					axios.post('${ctx}/signaling/isExistSignalingRelateFile.action',stringify({
						id: id
			        })).then(function(response){
						var data = response.data;
			            if(data["success"]){
			            	$("#winTest_gnb").window("open").window('center'); 
			            	$("#winTest_gnb").window("refresh", "${ctx}/signaling/toSignalingDetailInfo.action?id="+id);
			            }else{
			              	vm.$message.error('<%=rb.getString("XinLinXiaoXiBuCunZai")%>')
			            }
			    	})
				},
				//结果关闭
				cancelResultSlide(){
	        		this.$refs.resultSlide.hide();
	      		},
				//开始
				startTrace(row) {
          			var vm = this;

			        axios.post('${ctx}/signaling/startSignalingTraceTask.action',stringify({
			        	traceId : row.trace_id
			        })).then(function(response){
						var data = response.data;
			            if(data["success"]){
			            	vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type: 'success'
							});
			              	vm.$refs.tableTraceList.refresh()
			            }else{
			              	vm.$message.error(data["message"])
			            }
			    	})
        		},
        		//终止
        		terminateTrace(row) {
		        	var vm = this;
			    	    	
		          	axios.post('${ctx}/signaling/terminateSignalingTraceTask.action',stringify({
		          		traceId : row.trace_id
		          	})).then(function(response){
		            	var data = response.data;
			            if(data["success"]){
			            	vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type: 'success'
							});
			                vm.$refs.tableTraceList.refresh()
			            }else{
			                vm.$message.error(data["message"])
			            }
		          	})
		        },
		      	//重新开始
			    restartTrace(row){
					var vm = this;
					axios.post('${ctx}/signaling/restartSignalingTraceTask.action',stringify({
						traceId :  row.trace_id
		            })).then(function(response){
		                var data = response.data;
		                if(data["success"]){
		                	vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type: 'success'
							});
		                    vm.$refs.tableTraceList.refresh()
		                }else{
		                    vm.$message.error(data["message"])
		                }
		            })
				},
				//新建任务
				openChoseTraceTypeTask(){
          			var vm = this;
          			
          			vm.operateType = 'add';
          			vm.slideHeader = true;
			    	vm.slideFooter = true;
			    	vm.traceAddUrl = '${ctx}/signaling/toAddENBSignalingTracePage.action?type=gNB&timeZone=' + timeZone;
			    	vm.slidePosition = 'top';
			    	vm.slideHeight = '100%';
			    	vm.slideWidth = '100%';
                    vm.slideSubmitLoading = false;
			    	vm.slideTitle = '<%=rb.getString("eNBGenZongRenWuChuangJian")%>'.replace('eNB','gNB');
					vm.$refs.addTrace.showSlide(function(){
						vm.modal = false;
						eventBus.$emit('init-config-gnb','add', '');
				    });
	      		},
		        saveTrace() {
	      			var vm = this;
	      			eventBus.$emit('hander-ok-gnb')
		        },
		      	//关闭新建，详情
			    cancelAddTrace(){
		        	var vm = this;
		        	if(vm.operateType == 'view'){
		        		vm.$refs.addTrace.hide();
			    	}else{
			    		eventBus.$emit('close-slide-gnb')
			    	}
        		},
		      	//信息
				infoTrance(row){
			    	var vm = this;
			    	
			    	vm.operateType = 'view';
			    	vm.slideHeader = true;
			    	vm.slideFooter = false;
			    	vm.traceAddUrl = '${ctx}/signaling/querySignalingTraceProperties.action?traceId=' + row.trace_id + '&timeZone=' + timeZone;
			    	vm.slidePosition = 'top';
			    	vm.slideHeight = '100%';
			    	vm.slideWidth = '100%';
                    vm.slideSubmitLoading = false;
			    	vm.slideTitle = '<%=rb.getString("XinXi")%>';
					vm.$refs.addTrace.showSlide(function(){
						vm.modal = false;
						eventBus.$emit('init-config-gnb','readonly', row);
				    });
			    },
				//删除
		        deleteTrace(row) {
		        	var vm = this;
		
		            vm.$confirm('<%=rb.getString("QueRenShanChuRenWu")%>','<%=rb.getString("QueRen")%>',{
			            customClass:'warningConfirm',
			            confirmButtonText:'<%=rb.getString("QueDing")%>',
			            cancelButtonText:'<%=rb.getString("QuXiao")%>',
			            type:'warning',
			            closeOnClickModal:false
		          	}).then(() => {
			            axios.post('${ctx}/signaling/delSignalingTraceTask.action',stringify({
			            	traceId: row.trace_id
			            })).then(function(response){
							var data = response.data;
				            if(data["success"]){
				                vm.$message({
				                	type:'success',
				                    message:'<%=rb.getString("ChengGong")%>'
				                })
				                vm.$refs.tableTraceList.refresh();
								vm.$refs.resultSlide.hide();
				            }else{
				             	vm.$message.error(data["message"])
				            }
			            }).catch(function(error){
			              
			            })
		          	}).catch()
		        },
		        query(text) {
		          	var vm = this;
		
		          	Object.assign(vm.queryParams, {
		        	  	searchText: text,
		             	//startTime: '',
		              	//endTime: ''
		          	});
		        },
		        
				hideMenus() {
			        this.$refs.menu.hide()
			    },
			    hideTrace(){
					var vm = this;
			    	vm.$refs.addTrace.hide();
			    	vm.$refs.tableTraceList.refresh();
			    },
			    
			    dateChange(val) {
					var vm = this;
					vm.dateValue = val;
					if(val != null){
						vm.queryParams.startTime = vm.dateValue[0];
						vm.queryParams.endTime = vm.dateValue[1];
					}
				},
	    	},
	    	watch:{
				dateValue(newVal){
					var vm = this;
					if(!newVal){
						newVal = [];
						vm.queryParams.startTime = '';
		    			vm.queryParams.endTime = '';
					}
				},
			},
			mounted() {
		        var vm = this;
		        eventBus.$off("hander-cancel-gnb").$on('hander-cancel-gnb',this.hideTrace);
			}
		})
 	</script>
</body>
</html>