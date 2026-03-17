<%@ page import="java.util.Locale"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page contentType="text/html;charset=UTF-8"%>

<style>
	#imsi_ctn .el-tabs {
        width: 100%;
    }
    .npn-config .slide-content {
    	padding: 0px;
    }
    .normal-color::before {
    	color: #67D972;
    }
    .unusual-color::before {
    	color: #E88282;
    }
    .failure-color::before {
    	color: #F2B354;
    }
    .delete-color::before{
    	color:#CFCFCF;
    }
</style>

<div id="imsi_ctn" style="height: 100%;display: flex;">
    <el-tabs v-model="activeName" @tab-click="tabClick">
        <!-- IMSI Info -->
        <el-tab-pane label="IMSI Info" name="info">
            <el-ctable :url="imsiURL" :query-params="infoParams">
                <template slot="toolbar">
                    <div style="display: flex; justify-content: space-between;">
                        <el-query type="normal" @query="queryInfo" placeholder="IMSI"></el-query>
                        <div>
                            <i v-show="apnEnable" class="el-icon el-icon-circle-setting" style="margin-right: 10px;" @click="toAPN"></i>
                            <i class="el-icon el-icon-circle-export" style="margin-right: 10px;" @click="exportInfo"></i>
                        </div>
                    </div>
                </template>
                <el-table-column width="40" v-if="false">
                    <template slot-scope="scope">
                        <div class="el-icon el-icon-operation-more" @click="optClick(scope.row,event)" v-clickoutside="handerClose"></div>
                    </template>
                </el-table-column>
                <el-table-column label="<%=rb.getString("IMSI")%>" prop="imsi"></el-table-column>
                <el-table-column label="<%=rb.getString("JiHuoZhuangTai")%>" prop="activeStatus">
                	<template slot-scope="scope">
                		<span v-if="scope.row.activeStatus == 1" style="display: flex;"><i style="font-size: 20px;" class="el-icon el-icon-status-active1 normal-color"></i> <%=rb.getString("JiHuo")%></span>
                		<span v-else style="display: flex;"><i style="font-size: 20px;" class="el-icon el-icon-status-de-active unusual-color"></i> <%=rb.getString("WeiJiHuo")%></span>
                	</template>
                </el-table-column>
                <el-table-column label="<%=rb.getString("ZhuangTai")%>" prop="assignStatus">
                	<template slot-scope="scope">
                		<span v-if="scope.row.assignStatus == 1" style="display: flex;"><i style="font-size: 20px;" class="el-icon el-icon-status-SIM-inuse normal-color"></i> <%=rb.getString("YiFenPei")%></span>
                		<span v-else style="display: flex;"><i style="font-size: 20px;" class="el-icon el-icon-status-SIM-avaliable1"></i> <%=rb.getString("WeiFenPei")%></span>
                	</template>
                </el-table-column>
                <el-table-column label="<%=rb.getString("ZhuJian")%>" prop="key"></el-table-column>
                <el-table-column label="OPC" prop="opc"></el-table-column>
                <el-table-column v-if="apnEnable" label="<%=rb.getString("APNMingCheng")%>" prop="apnName">
                    <template slot-scope="scope">
                		<div v-if="scope.row.apnName && scope.row.apnName.split(',').length>0">
                            {{scope.row.apnName.split(',')[0]}} 
                            <el-popover trigger="click">
                                <span slot="reference" style="color: #4D84FF;cursor: pointer;" @click="queryIMSIApn(scope.row.imsi,scope.row)">[{{scope.row.apnName.split(',').length}}]</span>
                                <div v-for="item in getApns(scope.row)" style="display: flex;">
                                    <span style="display: flex;width: 160px;">
                                        <span style="display:inline-block;"><%=rb.getString("APNMingCheng")%><%=rb.getString("MaoHao")%> </span>
                                        <span style="white-space: nowrap;display:inline-block;overflow: hidden;text-overflow: ellipsis;max-width: 80px;padding-left: 2px;">{{item.apnName}}</span>
                                    </span>
                                    <span style="padding: 0 15px 0 10px;white-space: nowrap;"><%=rb.getString("JingTaiDongTaiIP")%><%=rb.getString("MaoHao")%> {{item.dyn==1?'Static':(item.dyn==0?'DYN':'')}}</span>
                                    <span style="white-space: nowrap;">IP<%=rb.getString("MaoHao")%> {{item.ip}}</span>
                                </div>
                            </el-popover>
                        </div>
                	</template>
                </el-table-column>
                <el-table-column v-if="false" label="<%=rb.getString("JingTaiDongTaiIP")%>" prop="dyn"></el-table-column>
                <el-table-column v-if="false" label="IP" prop="ip"></el-table-column>
                <el-table-column label="UE_DL_AMBR" prop="ue_dl_ambr"></el-table-column>
                <el-table-column label="UE_UL_AMBR" prop="ue_ul_ambr"></el-table-column>
            </el-ctable>

            <el-cmenu ref="menu" :data="menus"></el-cmenu>
        </el-tab-pane>
        <!-- IMSI eNB -->
        <el-tab-pane label="IMSI-eNB" name="enb">
            <el-ctable :url="imsiEnbURL" :query-params="enbParams">
                <template slot="toolbar">
                    <div style="display: flex; justify-content: space-between;">
                        <el-query type="normal" @query="queryEnb" placeholder="IMSI/<%=rb.getString("XiaoZhanXuLieHao")%>"></el-query>
                        <div>
                            <i class="el-icon el-icon-circle-export" style="margin-right: 10px;" @click="exportEnb"></i>
                        </div>
                    </div>
                </template>
                <el-table-column width="40" v-if="false">
                    <template slot-scope="scope">
                        <div class="el-icon el-icon-operation-more" v-clickoutside="handerClose"></div>
                    </template>
                </el-table-column>
                <el-table-column label="<%=rb.getString("XiaoZhanXuLieHao")%>" prop="sn"></el-table-column>
                <el-table-column label="<%=rb.getString("XiaoZhanECI")%>" prop="eci"></el-table-column>
                <el-table-column label="<%=rb.getString("IMSI")%>" prop="imsi"></el-table-column>
                <el-table-column label="<%=rb.getString("FenPeiZhuangTai")%>" prop="assignStatus">
                	<template slot-scope="scope">
                		<span v-if="scope.row.assignStatus == 0" style="display: flex;">
                			<i style="font-size: 20px;" class="el-icon el-icon-status-SIM-delete delete-color"></i> 
                			<%=rb.getString("YiShanChu")%>
                		</span>
                		<span v-if="scope.row.assignStatus == 1" style="display: flex;">
                			<i style="font-size: 20px;" class="el-icon el-icon-status-SIM-inuse normal-color"></i> 
                			<%=rb.getString("YiFenPei")%>
                		</span>
                		<span v-if="scope.row.assignStatus == 2" style="display: flex;">
                			<i style="font-size: 20px;" class="el-icon el-icon-status-SIM-failure unusual-color"></i> 
                			<%=rb.getString("FenPeiShiBai")%>
                		</span>
                	</template>
                </el-table-column>
                <el-table-column label="<%=rb.getString("ShiBaiYuanYin")%>" prop="reason"></el-table-column>
                <el-table-column label="<%=rb.getString("QingQiuShiJian")%>" prop="requestTime"></el-table-column>
                <el-table-column label="<%=rb.getString("XiangYingShiJian")%>" prop="responseTime"></el-table-column>
            </el-ctable>
        </el-tab-pane>
    </el-tabs>

    <el-slide ref="slide" :url="slideURL" method="get" :title="slideTitle" :header="false" class="npn-config" :footer="false"></el-slide>
</div>

<script>
new Vue({
    el: '#imsi_ctn',
    data() {

        return {
            activeName: 'info',
            slideURL: '',
            slideTitle: '',
            menus: [],
            imsiURL: '${ctx}/cell/imsi/queryImsiPageList.action',
            imsiEnbURL: '${ctx}/cell/imsi/queryImsiEnbPageList.action',
            infoParams: {
            	timeZone: timeZone,
            	searchText: '',
            	rd: ''
            },
            enbParams: {
            	timeZone: timeZone,
            	searchText: '',
            	rd: ''
            },
            apnEnable: false
        };
    },
    computed: {
    	apnShow() {
    		return writableMap.CODE_ENB_DEVICE_HALOB == true;
    	}
    },
    methods: {
    	getApnInfo() {
    		var vm = this;
    		
    		axios.post('${ctx}/sys/login/getFeatureCodesApnInfo.action').then(function(res){
    			var data = res.data;
    			
    			if(data) {
    				vm.apnEnable = data.permission == 'true';
    			}
    		}).catch(function(){});
    	},
        //IMSI Info 搜索事件
    	queryInfo(val){
    		Object.assign(this.infoParams,{
    			searchText: val,
    			rd: Math.random()
    		})
    	},
        //IMSI enb 搜索事件 
    	queryEnb(val){
    		Object.assign(this.enbParams,{
    			searchText: val,
    			rd: Math.random()
    		})
    	},
        getApns(row) {
            return row.rows||[];
        },
        queryIMSIApn(imsi,row) {
            var list = [],
                bool = row.popover;
            if(!bool) {
                $.ajax({
                    url: '${ctx}/cell/imsi/queryApnList.action',
                    type: 'post',
                    data: {imsi: imsi},
                    async: false,
                    dataType: 'json',
                    success: function(data){
                        if(data && data.rows) {
                        	list = data.rows;
                            row.rows = data.rows;
                        }
                        row.popover = true;
                    }
                })
            }else {
                list = row.rows
            }

            return list;
        },
        //IMSI Info 导出
    	exportInfo() {
    		exportByForm('${ctx}/cell/imsi/exportImsiList',this.infoParams);
    	},
        //IMSI enb 导出 
    	exportEnb() {
    		exportByForm('${ctx}/cell/imsi/exportImsiEnbList',this.enbParams);
    	},
        // tab 切换
        tabClick(tab,ev) {

        },
        // 点击页面其他地方关闭菜单
        handerClose(){
            this.$refs.menu.hide();
        },
        // 打开 APN页面
        toAPN() {
            var vm = this;

            vm.slideURL = '${ctx}/epc/apnconfig/toGwApnConfig.action';
            vm.$refs.slide.showSlide();
        },
        // 关闭 APN页面
        closeSlide() {
        	var vm = this;
        	
        	vm.$refs.slide.hide();
        },
        /**
        * 表格数据点击出现menus菜单事件
        * @param row{object}   行数据
        * @param ev{object}   event数据
        */
        optClick(row,ev) {
            var vm = this;

            vm.menus= [
                {label:'Active',cls:"el-icon-operation-active el-icon",code:'active'},
                {label:'Deactive',cls:"el-icon-operation-deactive el-icon" ,code:'deactive'},
            ];
            
            vm.$nextTick(function(){
                document.body.click();
                vm.$refs.menu.show(ev);
            });
        }
    },
    mounted() {
    	this.getApnInfo();
		eventBus.$off('close-imsi').$on('close-imsi',this.closeSlide);
    }
});
</script>
