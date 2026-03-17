<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>
<style>
	#mrFileDownloadPage{
		background-color: #FFFFFF;
		height: 100%;
		width: 100%;
		overflow: hidden;
		position: relative;
	}
	#mrFileDownloadPage .loading::before{
		background-color: rgba(255,255,255,1);
	}
	#mrFileDownloadPage .deviceRecycleBinHeaderCls{
		position:relative;
	}
	#mrFileDownloadPage .deviceListTableCls .el-ctable-toolbar{
		padding: 0px!important;
	}
	#mrFileDownloadPage .deviceListTableCls{
		width: 100%;
		height: 100%;
		overflow: auto;
		position:relative;
	}
    #mrFileDownloadPage .el-date-editor .el-range__close-icon {
        line-height:20px;
    }
</style>
<div id='mrFileDownloadPage'>
	<div class="container commonWarp" style="min-width: 900px;">
		<div class="deviceListTableCls">
			<el-ctable
				id="mrFileTable"
				ref="mrFileTable"
				:url="mrFileTableUrl"
				:height="height"
				:row-key="'id'"
				:query-params="queryFileParams"
				pagination="true"
				:rownumber=true
				@selection-change='batchSelect'
				>
                <template slot="toolbar">
                    <div class="deviceRecycleBinHeaderCls">
                        <div class="toolbarHeadBtnBoxCls">
                            <!-- 按钮   关闭 -->
                            <div class="newIconBoxCls-bt" style="position: absolute;right:20px" @click="closeDownloadFile" tip="<%=rb.getString("GuanBi")%>">		
                                <span class="el-icon-close el-icon"></span>
                            </div>
                            <div style="margin:0px 10px;font-size:14px;font-weight:bold;"><%=rb.getString("WenJianXiaZai")%></div>
                            <div class="selectBlukBoxCls">
                                <div class="selectMain">
                                    <div class="bulkSelectBtnBoxCls"  @click="openBulkSelectTable">
                                        <span class="el-icon-selected el-icon"></span>
                                        <span class="bulkSelectNumBoxCls">( {{mrFileSelection.length}} )</span>
                                    </div>
                                    <div class="selectTableBoxCls" style="position: absolute;top: 38px;left: 0px;" v-show="bulkSelectShow">
                                        <div class="selectBoxTitle">
                                            <span><%=rb.getString("YiXuan")%></span>
                                            <span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="closeBulkSelectTable"></span>
                                        </div>
                                        <div class="selectBoxMain">
                                            <div class="tableInfoCls">
                                                <div class="tableInfoHeader">
                                                    <div><%=rb.getString("WenJianMing")%></div>
                                                    <div @click="clearBulkSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span>Clear</div>
                                                </div>
                                                <el-ctable
                                                    id="bulkSelectTable"
                                                    ref="bulkSelectTable"
                                                    :data="mrFileSelection"
                                                    :showHeader="false"
                                                    :rownumber="false"
                                                    :front-pagination="true"
                                                    :row-key="'id'"
                                                    height="270px" pagination="true" >
                                                    <el-table-column prop="id" v-if="false"></el-table-column>
                                                    <el-table-column width="588">
                                                        <template slot-scope="scope" >
                                                            <div class="tableItemCls">
                                                                <span>{{scope.row.fileName}}</span>
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
                            <div :class="mrFileSelection.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" @click="downloadFile">
                                <span class="el-icon el-icon-operation-download"></span>
                                <span><%=rb.getString("XiaZai")%></span>
                            </div>
                        </div>
                        <div class="tableHeadBoxCls" style="position:relative;">
                            <div id="tableHeadQuery" class="tableHeadQueryBoxCls">
                                <div class="headQueryBox">
                                    <el-date-picker style='margin-left: 10px;width: 150px;' 
                                        key="queryTableDateDay"
                                        v-model="queryTableDay"
                                        type="date"
                                        value-format="yyyy-MM-dd"
                                        @change="queryTableDayChange"
                                        :clearable="false"
                                    >
                                    </el-date-picker>
                                    <el-time-picker style='margin-left: 10px;width: 240px;' 
                                        key="queryTableDateTime"
                                        is-range
                                        v-model="queryTableTime"
                                        value-format="HH:mm:ss"
                                        range-separator="——"  
                                        @change="queryTableTimeChange"
                                        :clearable="false"
                                        start-placeholder='<%=rb.getString("KaiShiShiJian")%>' 
                                        end-placeholder='<%=rb.getString("JieShuShiJian")%>'>
                                    </el-date-picker>
                                </div>
                            </div>
                        </div>
                    </div>
                </template>
				<!--设备列表-->
				<el-table-column width="50" type="selection" prop="ck"></el-table-column>
				<el-table-column key="fileName" prop="fileName" label='<%=rb.getString("WenJianMing")%>' min-width="120"></el-table-column>
				<!-- <el-table-column key="reportTime" label='<%=rb.getString("ShangBaoShiJian")%>' min-width="100" prop="reportTime"></el-table-column> -->
			</el-ctable>
		</div>
	</div>
</div>
<script type="text/javascript">
var mrFileDownloadVue = new Vue({
	el:'#mrFileDownloadPage',
	data(){
		return {
			bulkSelectShow:false,
			mrFileTableUrl:'',
			height:'100%',
            queryTableDay: '',
            queryTableTime:[],
			queryFileParams:{
                smallCellCode:'',
				startTime:'',
                endTime:'',
				timeZone:timeZone,
			},
			mrFileSelection:[],
		}
	},
    computed:{},
	methods:{
		// 初始化
		init(row){
			var vm = this;
			vm.smallCellCode = row.smallCellCode;
            vm.queryFileParams.smallCellCode = row.smallCellCode;
            vm.$nextTick(function () {
                if(row.startTime){
                    vm.queryTableDay = row.startTime.slice(0, 10);
                    vm.queryTableTime = ['00:00:00', '23:59:59'];
                    vm.queryFileParams.startTime = vm.queryTableDay + ' 00:00:00';
                    vm.queryFileParams.endTime = vm.queryTableDay + ' 23:59:59';
                }else{
                    let newDayTime = formatDate(new Date(gloableTime));
                    vm.queryTableDay = newDayTime.slice(0, 10);
                    vm.queryTableTime = ['00:00:00', '23:59:59'];
                    vm.queryFileParams.startTime = vm.queryTableDay + ' 00:00:00';
                    vm.queryFileParams.endTime = vm.queryTableDay + ' 23:59:59';
                }
                vm.mrFileTableUrl = '${ctx}/cell/perfmgmt/mrreport/getMRFilesTreeNodes.action';
            });
		},
		/**
		* 选择的批量数据
		* @param selection:传入批量数据对象
		*/
	    batchSelect(selection){
	    	var vm = this;
	    	vm.mrFileSelection = selection;
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

			vm.$refs.mrFileTable.clearSelection();
            vm.bulkSelectShow = false;
		},
		// 设备已选表格 单个删除事件
		delBulkSelected(rows){
			var vm = this,
				tabs = 'mrFileTable',
				rowKey = 'id';

			vm.mrFileSelection = vm.mrFileSelection.filter((items)=>{
				return items[rowKey] != rows[rowKey]
			});
			var selection = this.$refs[tabs].$refs.ctableInner.store.states.selection,
				irow= selection.filter((items)=>{
					return items[rowKey] == rows[rowKey]
				})[0];
			vm.$refs[tabs].toggleRowSelection(irow,false);
			var idx = vm.$refs[tabs].ckList.indexOf(rows[rowKey]);
			vm.$refs[tabs].ckList.splice(idx,1);
            if(vm.mrFileSelection.length == 0){
                vm.bulkSelectShow = false;
            }
		},
        // 日期选择器改变事件
        queryTableDayChange(val){
            var vm = this;
            vm.queryTableDay = val;
            if(val != null){
                vm.queryFileParams.startTime = val + ' ' + vm.queryTableTime[0];
                vm.queryFileParams.endTime = val + ' ' + vm.queryTableTime[1];
            }else{
                vm.queryFileParams.startTime = '';
                vm.queryFileParams.endTime = '';
            }
        },
        // 时间选择器改变事件
        queryTableTimeChange(val){
            var vm = this;
            vm.queryTableTime = val;
            if(val != null){
                vm.queryFileParams.startTime = vm.queryTableDay + ' ' + vm.queryTableTime[0];
                vm.queryFileParams.endTime = vm.queryTableDay + ' ' + vm.queryTableTime[1];
            }else{
                vm.queryFileParams.startTime = '';
                vm.queryFileParams.endTime = '';
            }
        },
		// 批量下载
		downloadFile(){
            var vm = this, 
                params = {
                    smallCellCode: vm.smallCellCode,
                    startTime: vm.queryFileParams.startTime,
                    endTime: vm.queryFileParams.endTime,
			        timeZone: timeZone
                };
            if(vm.mrFileSelection.length == 0){
                return
            }
            var mrFilesPath = '',mrFilesName = '';
            vm.mrFileSelection.map(function(item,idx){
				if(idx){
					mrFilesPath += ','+item.id
					mrFilesName +=  ','+item.fileName
				}else{
					mrFilesPath += item.id
					mrFilesName += item.fileName
				}
			});
            params.mrFilesPath = mrFilesPath;
            params.mrFilesName = mrFilesName;
            exportByForm("${ctx}/cell/perfmgmt/mrreport/downloadMRFiles.action", params);
            vm.$refs.mrFileTable.clearSelection();
        },
		closeDownloadFile(){
			var vm = this;
			mrPageVue.sharingSlideCancel();
		},
	},
	mounted(){
		var vm = this;
		eventBus.$off("mrFileDownload-init").$on("mrFileDownload-init",this.init)
	},

})

</script>
